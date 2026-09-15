package manual

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

func init() { provider.RegisterResource(NewAvailableVlanResource) }

var availableVlanStatusValues = []string{"active", "reserved", "deprecated"}

// AvailableVlanModel is the Terraform state of netbox_available_vlan.
type AvailableVlanModel struct {
	Id           types.Int64          `tfsdk:"id"`
	VlanGroupId  types.Int64          `tfsdk:"vlan_group_id"`
	Vid          types.Int64          `tfsdk:"vid"`
	Name         types.String         `tfsdk:"name"`
	TenantId     types.Int64          `tfsdk:"tenant_id"`
	Status       types.String         `tfsdk:"status"`
	RoleId       types.Int64          `tfsdk:"role_id"`
	Description  types.String         `tfsdk:"description"`
	Tags         types.Set            `tfsdk:"tags"`
	CustomFields jsontypes.Normalized `tfsdk:"custom_fields"`
	Url          types.String         `tfsdk:"url"`
	Display      types.String         `tfsdk:"display"`
}

var (
	_ resource.Resource                = &AvailableVlanResource{}
	_ resource.ResourceWithConfigure   = &AvailableVlanResource{}
	_ resource.ResourceWithImportState = &AvailableVlanResource{}
	_ resource.ResourceWithIdentity    = &AvailableVlanResource{}
)

// AvailableVlanResource allocates the next free VLAN ID of a VLAN group and
// then manages the VLAN like a netbox_vlan.
type AvailableVlanResource struct {
	client *netbox.APIClient
}

// NewAvailableVlanResource returns a new netbox_available_vlan resource.
func NewAvailableVlanResource() resource.Resource { return &AvailableVlanResource{} }

func (r *AvailableVlanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_available_vlan"
}

func (r *AvailableVlanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Allocates the next available VLAN ID in a VLAN group (`POST /api/ipam/vlan-groups/{id}/available-vlans/`) " +
			"and manages the resulting VLAN (`/api/ipam/vlans/`). Changing `vlan_group_id` allocates a new VLAN.\n\n" +
			"After `terraform import` the group can be read back from the VLAN, so `vlan_group_id` is populated.",
		Attributes: map[string]schema.Attribute{
			"id": idAttribute("The numeric ID of the VLAN in NetBox."),
			"vlan_group_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the VLAN group (`netbox_vlan_group`) to allocate from.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"vid": computedInt64Attribute("The allocated numeric VLAN ID (1-4094)."),
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the VLAN.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 64)},
			},
			"tenant_id": fkAttribute("ID of the Tenant (`netbox_tenant`)."),
			"status": choiceAttribute("Operational status of this VLAN. Valid values: `active`, `reserved`, `deprecated`. "+
				"Defaults to the NetBox server default when omitted.", availableVlanStatusValues),
			"role_id":       fkAttribute("ID of the VLAN role (`netbox_ipam_role`)."),
			"description":   blankStringAttribute("Description.", 200),
			"tags":          tagsAttribute(),
			"custom_fields": customFieldsAttribute(),
			"url":           computedStringAttribute("URL of the object in the NetBox API.", true),
			"display":       computedStringAttribute("Display name of the object.", false),
		},
	}
}

func (r *AvailableVlanResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = int64IdentitySchema()
}

func (r *AvailableVlanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *AvailableVlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AvailableVlanModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiPath := fmt.Sprintf("/api/ipam/vlan-groups/%d/available-vlans/", plan.VlanGroupId.ValueInt64())
	b := body{"name": plan.Name.ValueString()}
	availableVlanBody(ctx, b, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "allocating VLAN", map[string]any{"path": apiPath})
	obj, err := allocate[netbox.VLAN](ctx, r.client, apiPath, b)
	if err != nil {
		resp.Diagnostics.AddError("Error allocating netbox_available_vlan", err.Error())
		return
	}
	state := plan
	availableVlanFromAPI(obj, &plan, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	provider.SetIdentityID(ctx, resp.Identity, state.Id, &resp.Diagnostics)
}

func (r *AvailableVlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AvailableVlanModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	obj, res, err := r.client.IpamAPI.IpamVlansRetrieve(ctx, id).Execute()
	if err != nil {
		werr := netbox.WrapError(err, res)
		if netbox.IsNotFound(werr) {
			tflog.Info(ctx, "VLAN no longer exists, removing from state", map[string]any{"id": id})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading netbox_available_vlan", werr.Error())
		return
	}
	prior := state
	availableVlanFromAPI(obj, &prior, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	provider.SetIdentityID(ctx, resp.Identity, state.Id, &resp.Diagnostics)
}

func (r *AvailableVlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AvailableVlanModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	patch := netbox.NewPatchedWritableVLANRequest()
	patch.SetName(plan.Name.ValueString())
	if plan.TenantId.IsNull() {
		patch.SetTenantNil()
	} else if !plan.TenantId.IsUnknown() {
		patch.SetTenant(conv.Int32(plan.TenantId))
	}
	if conv.Known(plan.Status) {
		patch.SetStatus(plan.Status.ValueString())
	}
	if plan.RoleId.IsNull() {
		patch.SetRoleNil()
	} else if !plan.RoleId.IsUnknown() {
		patch.SetRole(conv.Int32(plan.RoleId))
	}
	if !plan.Description.IsUnknown() {
		patch.SetDescription(plan.Description.ValueString())
	}
	if conv.Known(plan.Tags) {
		patch.SetTags(conv.TagsToAPI(ctx, plan.Tags, &resp.Diagnostics))
	}
	if conv.Known(plan.CustomFields) {
		patch.SetCustomFields(conv.JSONObjectToAPI(plan.CustomFields, &resp.Diagnostics))
	}
	if resp.Diagnostics.HasError() {
		return
	}
	obj, res, err := r.client.IpamAPI.IpamVlansPartialUpdate(ctx, id).PatchedWritableVLANRequest(*patch).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating netbox_available_vlan", netbox.WrapError(err, res).Error())
		return
	}
	out := plan
	availableVlanFromAPI(obj, &plan, &out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
	provider.SetIdentityID(ctx, resp.Identity, out.Id, &resp.Diagnostics)
}

func (r *AvailableVlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AvailableVlanModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	res, err := r.client.IpamAPI.IpamVlansDestroy(ctx, id).Execute()
	if err != nil {
		if werr := netbox.WrapError(err, res); !netbox.IsNotFound(werr) {
			resp.Diagnostics.AddError("Error deleting netbox_available_vlan", werr.Error())
		}
	}
}

func (r *AvailableVlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	provider.ImportInt64ID(ctx, req, resp)
}

// availableVlanBody fills the allocation request from the plan. The VID and
// group are chosen by NetBox.
func availableVlanBody(ctx context.Context, b body, plan *AvailableVlanModel, diags *diag.Diagnostics) {
	b.nullableInt("tenant", plan.TenantId)
	b.str("status", plan.Status)
	b.nullableInt("role", plan.RoleId)
	b.str("description", plan.Description)
	b.tags(ctx, plan.Tags, diags)
	b.customFields(plan.CustomFields, diags)
}

// availableVlanFromAPI copies the API object into out. The group is readable
// from the VLAN, so vlan_group_id is always refreshed (imports included).
func availableVlanFromAPI(obj *netbox.VLAN, prior, out *AvailableVlanModel) {
	out.Id = types.Int64Value(int64(obj.GetId()))
	out.VlanGroupId = conv.BriefID(obj.GetGroupOk())
	out.Vid = conv.Int64From32(obj.GetVidOk())
	out.Name = conv.String(obj.GetNameOk())
	out.TenantId = conv.BriefID(obj.GetTenantOk())
	out.Status = conv.Choice(obj.GetStatusOk())
	out.RoleId = conv.BriefID(obj.GetRoleOk())
	out.Description = conv.StringOrEmpty(obj.GetDescriptionOk())
	out.Tags = conv.TagsFromAPI(obj.GetTags())
	out.CustomFields = conv.CustomFieldsFromAPI(obj.GetCustomFields(), prior.CustomFields)
	out.Url = conv.String(obj.GetUrlOk())
	out.Display = conv.String(obj.GetDisplayOk())
}
