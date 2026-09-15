package manual

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	"github.com/elliot/terraform-provider-netbox/internal/customfields"
	"github.com/elliot/terraform-provider-netbox/internal/provider"
	"github.com/elliot/terraform-provider-netbox/netbox"
)

func init() { provider.RegisterResource(NewAvailableIpAddressResource) }

var (
	availableIpAddressStatusValues = []string{"active", "reserved", "deprecated", "dhcp", "slaac"}
	availableIpAddressRoleValues   = []string{"loopback", "secondary", "anycast", "vip", "vrrp", "hsrp", "glbp", "carp"}
	dnsNameRegexp                  = regexp.MustCompile(`^$|^([0-9A-Za-z_-]+|\*)(\.[0-9A-Za-z_-]+)*\.?$`)
)

// AvailableIpAddressModel is the Terraform state of netbox_available_ip_address.
type AvailableIpAddressModel struct {
	Id                 types.Int64   `tfsdk:"id"`
	PrefixId           types.Int64   `tfsdk:"prefix_id"`
	IpRangeId          types.Int64   `tfsdk:"ip_range_id"`
	Address            types.String  `tfsdk:"address"`
	VrfId              types.Int64   `tfsdk:"vrf_id"`
	TenantId           types.Int64   `tfsdk:"tenant_id"`
	Status             types.String  `tfsdk:"status"`
	Role               types.String  `tfsdk:"role"`
	AssignedObjectType types.String  `tfsdk:"assigned_object_type"`
	AssignedObjectId   types.Int64   `tfsdk:"assigned_object_id"`
	DnsName            types.String  `tfsdk:"dns_name"`
	Description        types.String  `tfsdk:"description"`
	Comments           types.String  `tfsdk:"comments"`
	Tags               types.Set     `tfsdk:"tags"`
	CustomFields       types.Dynamic `tfsdk:"custom_fields"`
	Url                types.String  `tfsdk:"url"`
	Display            types.String  `tfsdk:"display"`
}

var (
	_ resource.Resource                = &AvailableIpAddressResource{}
	_ resource.ResourceWithConfigure   = &AvailableIpAddressResource{}
	_ resource.ResourceWithImportState = &AvailableIpAddressResource{}
	_ resource.ResourceWithIdentity    = &AvailableIpAddressResource{}
)

// AvailableIpAddressResource allocates the next free IP address of a prefix
// or an IP range and then manages it like a netbox_ip_address.
type AvailableIpAddressResource struct {
	client *netbox.APIClient
	cf     *customfields.Cache
}

// NewAvailableIpAddressResource returns a new netbox_available_ip_address resource.
func NewAvailableIpAddressResource() resource.Resource { return &AvailableIpAddressResource{} }

func (r *AvailableIpAddressResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_available_ip_address"
}

func (r *AvailableIpAddressResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Allocates the next available IP address from a prefix (`POST /api/ipam/prefixes/{id}/available-ips/`) " +
			"or an IP range (`POST /api/ipam/ip-ranges/{id}/available-ips/`) and manages the resulting IP address " +
			"(`/api/ipam/ip-addresses/`). Exactly one of `prefix_id` and `ip_range_id` must be set; changing either " +
			"allocates a new address. The VRF is inherited from the parent.\n\n" +
			"After `terraform import` the parent is unknown, so `prefix_id` / `ip_range_id` stay null in state; " +
			"setting one in the configuration of an imported resource forces a replacement.",
		Attributes: map[string]schema.Attribute{
			"id": idAttribute("The numeric ID of the IP address in NetBox."),
			"prefix_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the prefix (`netbox_prefix`) to allocate from. Conflicts with `ip_range_id`.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				Validators: []validator.Int64{
					int64validator.ExactlyOneOf(path.MatchRoot("prefix_id"), path.MatchRoot("ip_range_id")),
				},
			},
			"ip_range_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the IP range (`netbox_ip_range`) to allocate from. Conflicts with `prefix_id`.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"address":   computedStringAttribute("The allocated address in CIDR notation (for example `10.0.0.5/24`).", true),
			"vrf_id":    computedInt64Attribute("ID of the VRF (`netbox_vrf`), inherited from the parent prefix or range."),
			"tenant_id": fkAttribute("ID of the Tenant (`netbox_tenant`)."),
			"status": choiceAttribute("The operational status of this IP. Valid values: `active`, `reserved`, `deprecated`, `dhcp`, `slaac`. "+
				"Defaults to the NetBox server default when omitted.", availableIpAddressStatusValues),
			"role": nullableChoiceAttribute("The functional role of this IP. Valid values: `loopback`, `secondary`, `anycast`, `vip`, `vrrp`, `hsrp`, `glbp`, `carp`.",
				availableIpAddressRoleValues),
			"assigned_object_type": schema.StringAttribute{
				MarkdownDescription: "Content type of the object the address is assigned to: `dcim.interface`, `virtualization.vminterface` or `dcim.fhrpgroup`.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.AlsoRequires(path.MatchRoot("assigned_object_id"))},
			},
			"assigned_object_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the object the address is assigned to (for example a `netbox_interface`).",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.AlsoRequires(path.MatchRoot("assigned_object_type"))},
			},
			"dns_name": blankStringAttribute("Hostname or FQDN (not case-sensitive).", 255,
				stringvalidator.RegexMatches(dnsNameRegexp, "must be a hostname or FQDN")),
			"description":   blankStringAttribute("Description.", 200),
			"comments":      blankStringAttribute("Comments.", 0),
			"tags":          tagsAttribute(),
			"custom_fields": customFieldsAttribute(),
			"url":           computedStringAttribute("URL of the object in the NetBox API.", true),
			"display":       computedStringAttribute("Display name of the object.", false),
		},
	}
}

func (r *AvailableIpAddressResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = int64IdentitySchema()
}

func (r *AvailableIpAddressResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
	r.cf = configureCache(req)
}

func (r *AvailableIpAddressResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AvailableIpAddressModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var apiPath string
	switch {
	case conv.Known(plan.PrefixId):
		apiPath = fmt.Sprintf("/api/ipam/prefixes/%d/available-ips/", plan.PrefixId.ValueInt64())
	case conv.Known(plan.IpRangeId):
		apiPath = fmt.Sprintf("/api/ipam/ip-ranges/%d/available-ips/", plan.IpRangeId.ValueInt64())
	default:
		resp.Diagnostics.AddError("Missing parent", "One of prefix_id or ip_range_id must be set.")
		return
	}
	b := body{}
	availableIpAddressBody(ctx, b, &plan, r.cf, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "allocating IP address", map[string]any{"path": apiPath})
	obj, err := allocate[netbox.IPAddress](ctx, r.client, apiPath, b)
	if err != nil {
		resp.Diagnostics.AddError("Error allocating netbox_available_ip_address", err.Error())
		return
	}
	state := plan
	availableIpAddressFromAPI(ctx, obj, &plan, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	provider.SetIdentityID(ctx, resp.Identity, state.Id, &resp.Diagnostics)
}

func (r *AvailableIpAddressResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AvailableIpAddressModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	obj, res, err := r.client.IpamAPI.IpamIpAddressesRetrieve(ctx, id).Execute()
	if err != nil {
		werr := netbox.WrapError(err, res)
		if netbox.IsNotFound(werr) {
			tflog.Info(ctx, "IP address no longer exists, removing from state", map[string]any{"id": id})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading netbox_available_ip_address", werr.Error())
		return
	}
	prior := state
	availableIpAddressFromAPI(ctx, obj, &prior, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	provider.SetIdentityID(ctx, resp.Identity, state.Id, &resp.Diagnostics)
}

func (r *AvailableIpAddressResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AvailableIpAddressModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	patch := netbox.NewPatchedWritableIPAddressRequest()
	if plan.TenantId.IsNull() {
		patch.SetTenantNil()
	} else if !plan.TenantId.IsUnknown() {
		patch.SetTenant(conv.Int32(plan.TenantId))
	}
	if conv.Known(plan.Status) {
		patch.SetStatus(plan.Status.ValueString())
	}
	if plan.Role.IsNull() {
		patch.SetRoleNil()
	} else if !plan.Role.IsUnknown() {
		patch.SetRole(plan.Role.ValueString())
	}
	if plan.AssignedObjectType.IsNull() {
		patch.SetAssignedObjectTypeNil()
	} else if !plan.AssignedObjectType.IsUnknown() {
		patch.SetAssignedObjectType(plan.AssignedObjectType.ValueString())
	}
	if plan.AssignedObjectId.IsNull() {
		patch.SetAssignedObjectIdNil()
	} else if !plan.AssignedObjectId.IsUnknown() {
		patch.SetAssignedObjectId(plan.AssignedObjectId.ValueInt64())
	}
	if !plan.DnsName.IsUnknown() {
		patch.SetDnsName(plan.DnsName.ValueString())
	}
	if !plan.Description.IsUnknown() {
		patch.SetDescription(plan.Description.ValueString())
	}
	if !plan.Comments.IsUnknown() {
		patch.SetComments(plan.Comments.ValueString())
	}
	if conv.Known(plan.Tags) {
		patch.SetTags(conv.TagsToAPI(ctx, plan.Tags, &resp.Diagnostics))
	}
	if conv.Known(plan.CustomFields) {
		patch.SetCustomFields(customfields.ToAPI(ctx, plan.CustomFields, r.cf, "ipam.ipaddress", &resp.Diagnostics))
	}
	if resp.Diagnostics.HasError() {
		return
	}
	obj, res, err := r.client.IpamAPI.IpamIpAddressesPartialUpdate(ctx, id).PatchedWritableIPAddressRequest(*patch).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating netbox_available_ip_address", netbox.WrapError(err, res).Error())
		return
	}
	out := plan
	// The parent is not readable from the IP address; keep whatever state has.
	out.PrefixId, out.IpRangeId = state.PrefixId, state.IpRangeId
	availableIpAddressFromAPI(ctx, obj, &plan, &out, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
	provider.SetIdentityID(ctx, resp.Identity, out.Id, &resp.Diagnostics)
}

func (r *AvailableIpAddressResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AvailableIpAddressModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	res, err := r.client.IpamAPI.IpamIpAddressesDestroy(ctx, id).Execute()
	if err != nil {
		if werr := netbox.WrapError(err, res); !netbox.IsNotFound(werr) {
			resp.Diagnostics.AddError("Error deleting netbox_available_ip_address", werr.Error())
		}
	}
}

func (r *AvailableIpAddressResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	provider.ImportInt64ID(ctx, req, resp)
}

// availableIpAddressBody fills the allocation request from the plan. The VRF
// and the address itself are chosen by NetBox.
func availableIpAddressBody(ctx context.Context, b body, plan *AvailableIpAddressModel, cf *customfields.Cache, diags *diag.Diagnostics) {
	b.nullableInt("tenant", plan.TenantId)
	b.str("status", plan.Status)
	b.nullableStr("role", plan.Role)
	b.nullableStr("assigned_object_type", plan.AssignedObjectType)
	b.nullableInt("assigned_object_id", plan.AssignedObjectId)
	b.str("dns_name", plan.DnsName)
	b.str("description", plan.Description)
	b.str("comments", plan.Comments)
	b.tags(ctx, plan.Tags, diags)
	b.customFields(ctx, plan.CustomFields, cf, "ipam.ipaddress", diags)
}

// availableIpAddressFromAPI copies the API object into out. prior carries the
// previous state or plan (for custom field key selection).
func availableIpAddressFromAPI(ctx context.Context, obj *netbox.IPAddress, prior, out *AvailableIpAddressModel, diags *diag.Diagnostics) {
	out.Id = types.Int64Value(int64(obj.GetId()))
	out.Address = conv.String(obj.GetAddressOk())
	out.VrfId = conv.BriefID(obj.GetVrfOk())
	out.TenantId = conv.BriefID(obj.GetTenantOk())
	out.Status = conv.Choice(obj.GetStatusOk())
	out.Role = conv.Choice(obj.GetRoleOk())
	out.AssignedObjectType = conv.String(obj.GetAssignedObjectTypeOk())
	out.AssignedObjectId = conv.Int64From64(obj.GetAssignedObjectIdOk())
	out.DnsName = conv.StringOrEmpty(obj.GetDnsNameOk())
	out.Description = conv.StringOrEmpty(obj.GetDescriptionOk())
	out.Comments = conv.StringOrEmpty(obj.GetCommentsOk())
	out.Tags = conv.TagsFromAPI(obj.GetTags())
	out.CustomFields = customfields.FromAPI(ctx, obj.GetCustomFields(), prior.CustomFields, diags)
	out.Url = conv.String(obj.GetUrlOk())
	out.Display = conv.String(obj.GetDisplayOk())
}
