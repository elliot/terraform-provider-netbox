package manual

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/elliot/terraform-provider-netbox/internal/conv"
	"github.com/elliot/terraform-provider-netbox/internal/provider"
	"github.com/elliot/terraform-provider-netbox/netbox"
)

func init() {
	provider.RegisterResource(NewDevicePrimaryIpResource)
	provider.RegisterResource(NewVirtualMachinePrimaryIpResource)
}

// primaryIPKind describes the parent object type of a primary IP resource.
type primaryIPKind struct {
	// suffix is the Terraform type name without the provider prefix.
	suffix string
	// label names the parent in diagnostics and docs ("device").
	label string
	// parentAttr is the attribute holding the parent ID ("device_id").
	parentAttr string
	// parentResource is the Terraform type of the parent ("netbox_device").
	parentResource string
	// apiPath is the collection path ("/api/dcim/devices/").
	apiPath string
}

var (
	deviceKind = primaryIPKind{
		suffix: "_device_primary_ip", label: "device", parentAttr: "device_id",
		parentResource: "netbox_device", apiPath: "/api/dcim/devices/",
	}
	virtualMachineKind = primaryIPKind{
		suffix: "_virtual_machine_primary_ip", label: "virtual machine", parentAttr: "virtual_machine_id",
		parentResource: "netbox_virtual_machine", apiPath: "/api/virtualization/virtual-machines/",
	}
)

// primaryIpModel is the Terraform state of netbox_device_primary_ip and
// netbox_virtual_machine_primary_ip. ParentId is bound to `device_id` or
// `virtual_machine_id` depending on the resource, so the model is read and
// written attribute by attribute.
type primaryIpModel struct {
	Id               types.Int64
	ParentId         types.Int64
	IpAddressId      types.Int64
	IpAddressVersion types.Int64
}

// attrStore is implemented by tfsdk.Plan and tfsdk.State.
type attrStore interface {
	GetAttribute(ctx context.Context, p path.Path, target any) diag.Diagnostics
}

type attrSetter interface {
	SetAttribute(ctx context.Context, p path.Path, val any) diag.Diagnostics
}

func (r *PrimaryIpResource) getModel(ctx context.Context, src attrStore, m *primaryIpModel) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.Append(src.GetAttribute(ctx, path.Root("id"), &m.Id)...)
	diags.Append(src.GetAttribute(ctx, path.Root(r.kind.parentAttr), &m.ParentId)...)
	diags.Append(src.GetAttribute(ctx, path.Root("ip_address_id"), &m.IpAddressId)...)
	diags.Append(src.GetAttribute(ctx, path.Root("ip_address_version"), &m.IpAddressVersion)...)
	return diags
}

func (r *PrimaryIpResource) setModel(ctx context.Context, dst attrSetter, m *primaryIpModel) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.Append(dst.SetAttribute(ctx, path.Root("id"), m.Id)...)
	diags.Append(dst.SetAttribute(ctx, path.Root(r.kind.parentAttr), m.ParentId)...)
	diags.Append(dst.SetAttribute(ctx, path.Root("ip_address_id"), m.IpAddressId)...)
	diags.Append(dst.SetAttribute(ctx, path.Root("ip_address_version"), m.IpAddressVersion)...)
	return diags
}

// primaryIPParent is the subset of a device / virtual machine we read back.
type primaryIPParent struct {
	Id         int64 `json:"id"`
	PrimaryIp4 *struct {
		Id int64 `json:"id"`
	} `json:"primary_ip4"`
	PrimaryIp6 *struct {
		Id int64 `json:"id"`
	} `json:"primary_ip6"`
}

// primary returns the ID of the primary address of the given family, or null.
func (p *primaryIPParent) primary(version int64) types.Int64 {
	switch {
	case version == 4 && p.PrimaryIp4 != nil:
		return types.Int64Value(p.PrimaryIp4.Id)
	case version == 6 && p.PrimaryIp6 != nil:
		return types.Int64Value(p.PrimaryIp6.Id)
	}
	return types.Int64Null()
}

var (
	_ resource.Resource                = &PrimaryIpResource{}
	_ resource.ResourceWithConfigure   = &PrimaryIpResource{}
	_ resource.ResourceWithImportState = &PrimaryIpResource{}
	_ resource.ResourceWithIdentity    = &PrimaryIpResource{}
	_ resource.ResourceWithModifyPlan  = &PrimaryIpResource{}
)

// PrimaryIpResource sets the primary IPv4 or IPv6 address of a device or a
// virtual machine. Keeping the assignment out of the parent resource breaks
// the parent -> interface -> IP address -> parent dependency cycle.
type PrimaryIpResource struct {
	kind   primaryIPKind
	client *netbox.APIClient
}

// NewDevicePrimaryIpResource returns a new netbox_device_primary_ip resource.
func NewDevicePrimaryIpResource() resource.Resource { return &PrimaryIpResource{kind: deviceKind} }

// NewVirtualMachinePrimaryIpResource returns a new netbox_virtual_machine_primary_ip resource.
func NewVirtualMachinePrimaryIpResource() resource.Resource {
	return &PrimaryIpResource{kind: virtualMachineKind}
}

func (r *PrimaryIpResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.kind.suffix
}

func (r *PrimaryIpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	k := r.kind
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf("Sets the primary IPv4 or IPv6 address of a %[1]s (`PATCH %[2]s{id}/` with `primary_ip4` / `primary_ip6`). "+
			"Use it instead of `primary_ip4_id` / `primary_ip6_id` on `%[3]s` when the IP address is assigned to an interface of the same %[1]s, "+
			"which would otherwise create a dependency cycle (%[1]s -> interface -> IP address -> %[1]s).\n\n"+
			"NetBox requires the address to be assigned to an interface of the %[1]s. The resource ID is the %[1]s ID; "+
			"`terraform import netbox%[4]s.example <%[5]s>` picks the primary IPv4 when set, otherwise the primary IPv6. "+
			"Destroying the resource clears the primary address on the %[1]s.\n\n"+
			"The `%[3]s` resource also tracks `primary_ip4_id` / `primary_ip6_id`, so add "+
			"`lifecycle { ignore_changes = [primary_ip4_id, primary_ip6_id] }` to the `%[3]s` managing the parent; "+
			"otherwise its next refresh reports the assignment made here as drift and its next update clears it "+
			"(which this resource would then detect and repair).",
			k.label, k.apiPath, k.parentResource, k.suffix, k.parentAttr),
		Attributes: map[string]schema.Attribute{
			"id": idAttribute(fmt.Sprintf("The numeric ID of the %s (same as `%s`).", k.label, k.parentAttr)),
			k.parentAttr: schema.Int64Attribute{
				MarkdownDescription: fmt.Sprintf("ID of the %s (`%s`).", k.label, k.parentResource),
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"ip_address_id": schema.Int64Attribute{
				MarkdownDescription: fmt.Sprintf("ID of the IP address (`netbox_ip_address`) to set as primary. It must be assigned to an interface of the %s.", k.label),
				Required:            true,
			},
			"ip_address_version": schema.Int64Attribute{
				MarkdownDescription: "IP version of the address, `4` or `6`. Detected from the IP address when omitted.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.Int64{int64validator.OneOf(4, 6)},
			},
		},
	}
}

func (r *PrimaryIpResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = int64IdentitySchema()
}

func (r *PrimaryIpResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

// ModifyPlan keeps the detected ip_address_version when the address does not
// change; when it does, the version stays unknown because the new address may
// belong to the other family.
func (r *PrimaryIpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var plan, state primaryIpModel
	resp.Diagnostics.Append(r.getModel(ctx, req.Plan, &plan)...)
	resp.Diagnostics.Append(r.getModel(ctx, req.State, &state)...)
	if resp.Diagnostics.HasError() || !plan.IpAddressVersion.IsUnknown() {
		return
	}
	if plan.IpAddressId.Equal(state.IpAddressId) && conv.Known(state.IpAddressVersion) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("ip_address_version"), state.IpAddressVersion)...)
	}
}

// detectVersion returns the configured IP version or looks it up from the IP
// address's family.
func (r *PrimaryIpResource) detectVersion(ctx context.Context, m *primaryIpModel, diags *diag.Diagnostics) (int64, bool) {
	if conv.Known(m.IpAddressVersion) {
		return m.IpAddressVersion.ValueInt64(), true
	}
	id, ok := int32ID(m.IpAddressId, diags)
	if !ok {
		return 0, false
	}
	ip, res, err := r.client.IpamAPI.IpamIpAddressesRetrieve(ctx, id).Execute()
	if err != nil {
		diags.AddError("Error reading IP address", fmt.Sprintf("Cannot detect the IP version of IP address %d: %s", id, netbox.WrapError(err, res).Error()))
		return 0, false
	}
	family := conv.ChoiceInt(ip.GetFamilyOk())
	if family.ValueInt64() != 4 && family.ValueInt64() != 6 {
		diags.AddError("Unexpected IP family", fmt.Sprintf("IP address %d reports family %v; set ip_address_version explicitly.", id, family))
		return 0, false
	}
	return family.ValueInt64(), true
}

func (r *PrimaryIpResource) parentPath(m *primaryIpModel) string {
	return fmt.Sprintf("%s%d/", r.kind.apiPath, m.ParentId.ValueInt64())
}

// assign PATCHes the parent so that the primary address of newVersion is
// ipID (nil clears it) and, when oldVersion differs, clears the old family.
func (r *PrimaryIpResource) assign(ctx context.Context, m *primaryIpModel, newVersion int64, ipID *int64, oldVersion int64) (*primaryIPParent, error) {
	b := body{}
	if ipID == nil {
		b[fmt.Sprintf("primary_ip%d", newVersion)] = nil
	} else {
		b[fmt.Sprintf("primary_ip%d", newVersion)] = *ipID
	}
	if oldVersion != 0 && oldVersion != newVersion {
		b[fmt.Sprintf("primary_ip%d", oldVersion)] = nil
	}
	tflog.Debug(ctx, "setting primary IP", map[string]any{"kind": r.kind.label, "path": r.parentPath(m), "body": b})
	var out primaryIPParent
	if err := r.client.PatchRaw(ctx, r.parentPath(m), b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *PrimaryIpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan primaryIpModel
	resp.Diagnostics.Append(r.getModel(ctx, req.Plan, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	version, ok := r.detectVersion(ctx, &plan, &resp.Diagnostics)
	if !ok {
		return
	}
	ipID := plan.IpAddressId.ValueInt64()
	parent, err := r.assign(ctx, &plan, version, &ipID, 0)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error setting primary IP of %s", r.kind.label), err.Error())
		return
	}
	state := primaryIpModel{
		Id:               types.Int64Value(parent.Id),
		ParentId:         plan.ParentId,
		IpAddressId:      parent.primary(version),
		IpAddressVersion: types.Int64Value(version),
	}
	resp.Diagnostics.Append(r.setModel(ctx, &resp.State, &state)...)
	provider.SetIdentityID(ctx, resp.Identity, state.Id, &resp.Diagnostics)
}

func (r *PrimaryIpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state primaryIpModel
	resp.Diagnostics.Append(r.getModel(ctx, req.State, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !conv.Known(state.ParentId) {
		// Imported by parent ID.
		state.ParentId = state.Id
	}
	var parent primaryIPParent
	if err := r.client.GetRaw(ctx, r.parentPath(&state), &parent); err != nil {
		if netbox.IsNotFound(err) {
			tflog.Info(ctx, "parent no longer exists, removing primary IP from state", map[string]any{"kind": r.kind.label, "id": state.ParentId.ValueInt64()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("Error reading %s", r.kind.label), err.Error())
		return
	}
	version := state.IpAddressVersion.ValueInt64()
	if !conv.Known(state.IpAddressVersion) {
		// Import: prefer IPv4, fall back to IPv6.
		switch {
		case parent.PrimaryIp4 != nil:
			version = 4
		case parent.PrimaryIp6 != nil:
			version = 6
		}
	}
	current := parent.primary(version)
	if current.IsNull() {
		tflog.Info(ctx, "primary IP is no longer set, removing from state", map[string]any{"kind": r.kind.label, "id": parent.Id})
		resp.State.RemoveResource(ctx)
		return
	}
	state.Id = types.Int64Value(parent.Id)
	state.IpAddressId = current
	state.IpAddressVersion = types.Int64Value(version)
	resp.Diagnostics.Append(r.setModel(ctx, &resp.State, &state)...)
	provider.SetIdentityID(ctx, resp.Identity, state.Id, &resp.Diagnostics)
}

func (r *PrimaryIpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state primaryIpModel
	resp.Diagnostics.Append(r.getModel(ctx, req.Plan, &plan)...)
	resp.Diagnostics.Append(r.getModel(ctx, req.State, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	version, ok := r.detectVersion(ctx, &plan, &resp.Diagnostics)
	if !ok {
		return
	}
	ipID := plan.IpAddressId.ValueInt64()
	parent, err := r.assign(ctx, &plan, version, &ipID, state.IpAddressVersion.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error updating primary IP of %s", r.kind.label), err.Error())
		return
	}
	out := primaryIpModel{
		Id:               types.Int64Value(parent.Id),
		ParentId:         plan.ParentId,
		IpAddressId:      parent.primary(version),
		IpAddressVersion: types.Int64Value(version),
	}
	resp.Diagnostics.Append(r.setModel(ctx, &resp.State, &out)...)
	provider.SetIdentityID(ctx, resp.Identity, out.Id, &resp.Diagnostics)
}

func (r *PrimaryIpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state primaryIpModel
	resp.Diagnostics.Append(r.getModel(ctx, req.State, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	version := state.IpAddressVersion.ValueInt64()
	if version != 4 && version != 6 {
		return
	}
	if _, err := r.assign(ctx, &state, version, nil, 0); err != nil && !netbox.IsNotFound(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Error clearing primary IP of %s", r.kind.label), err.Error())
	}
}

func (r *PrimaryIpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	provider.ImportInt64ID(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	var id types.Int64
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(r.kind.parentAttr), id)...)
}
