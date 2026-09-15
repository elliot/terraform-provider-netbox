package manual

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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

func init() { provider.RegisterResource(NewAvailablePrefixResource) }

var availablePrefixStatusValues = []string{"container", "active", "reserved", "deprecated"}

// AvailablePrefixModel is the Terraform state of netbox_available_prefix.
type AvailablePrefixModel struct {
	Id             types.Int64   `tfsdk:"id"`
	ParentPrefixId types.Int64   `tfsdk:"parent_prefix_id"`
	PrefixLength   types.Int64   `tfsdk:"prefix_length"`
	Prefix         types.String  `tfsdk:"prefix"`
	VrfId          types.Int64   `tfsdk:"vrf_id"`
	ScopeType      types.String  `tfsdk:"scope_type"`
	ScopeId        types.Int64   `tfsdk:"scope_id"`
	TenantId       types.Int64   `tfsdk:"tenant_id"`
	Status         types.String  `tfsdk:"status"`
	RoleId         types.Int64   `tfsdk:"role_id"`
	IsPool         types.Bool    `tfsdk:"is_pool"`
	MarkUtilized   types.Bool    `tfsdk:"mark_utilized"`
	Description    types.String  `tfsdk:"description"`
	Comments       types.String  `tfsdk:"comments"`
	Tags           types.Set     `tfsdk:"tags"`
	CustomFields   types.Dynamic `tfsdk:"custom_fields"`
	Url            types.String  `tfsdk:"url"`
	Display        types.String  `tfsdk:"display"`
}

var (
	_ resource.Resource                = &AvailablePrefixResource{}
	_ resource.ResourceWithConfigure   = &AvailablePrefixResource{}
	_ resource.ResourceWithImportState = &AvailablePrefixResource{}
	_ resource.ResourceWithIdentity    = &AvailablePrefixResource{}
)

// AvailablePrefixResource allocates the next free child prefix of a parent
// prefix and then manages it like a netbox_prefix.
type AvailablePrefixResource struct {
	client *netbox.APIClient
	cf     *customfields.Cache
}

// NewAvailablePrefixResource returns a new netbox_available_prefix resource.
func NewAvailablePrefixResource() resource.Resource { return &AvailablePrefixResource{} }

func (r *AvailablePrefixResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_available_prefix"
}

func (r *AvailablePrefixResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Allocates the next available child prefix of the given length inside a parent prefix " +
			"(`POST /api/ipam/prefixes/{id}/available-prefixes/`) and manages the resulting prefix (`/api/ipam/prefixes/`). " +
			"The VRF is inherited from the parent. Changing `parent_prefix_id` or `prefix_length` allocates a new prefix.\n\n" +
			"After `terraform import` the parent and requested length are unknown, so `parent_prefix_id` and `prefix_length` " +
			"stay null in state; setting them in the configuration of an imported resource forces a replacement.",
		Attributes: map[string]schema.Attribute{
			"id": idAttribute("The numeric ID of the prefix in NetBox."),
			"parent_prefix_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the parent prefix (`netbox_prefix`) to allocate from.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"prefix_length": schema.Int64Attribute{
				MarkdownDescription: "Length of the prefix to allocate (1-32 for IPv4, 1-128 for IPv6). Must be longer than the parent prefix.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				Validators:          []validator.Int64{int64validator.Between(1, 128)},
			},
			"prefix": computedStringAttribute("The allocated prefix in CIDR notation.", true),
			"vrf_id": computedInt64Attribute("ID of the VRF (`netbox_vrf`), inherited from the parent prefix."),
			"scope_type": schema.StringAttribute{
				MarkdownDescription: "Content type of the scope object: `dcim.region`, `dcim.sitegroup`, `dcim.site` or `dcim.location`.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.AlsoRequires(path.MatchRoot("scope_id"))},
			},
			"scope_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the scope object (for example a `netbox_site`).",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.AlsoRequires(path.MatchRoot("scope_type"))},
			},
			"tenant_id": fkAttribute("ID of the Tenant (`netbox_tenant`)."),
			"status": choiceAttribute("Operational status of this prefix. Valid values: `container`, `active`, `reserved`, `deprecated`. "+
				"Defaults to the NetBox server default when omitted.", availablePrefixStatusValues),
			"role_id": fkAttribute("ID of the prefix role (`netbox_ipam_role`)."),
			"is_pool": schema.BoolAttribute{
				MarkdownDescription: "All IP addresses within this prefix are considered usable. Defaults to the NetBox server default when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"mark_utilized": schema.BoolAttribute{
				MarkdownDescription: "Treat as fully utilized. Defaults to the NetBox server default when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"description":   blankStringAttribute("Description.", 200),
			"comments":      blankStringAttribute("Comments.", 0),
			"tags":          tagsAttribute(),
			"custom_fields": customFieldsAttribute(),
			"url":           computedStringAttribute("URL of the object in the NetBox API.", true),
			"display":       computedStringAttribute("Display name of the object.", false),
		},
	}
}

func (r *AvailablePrefixResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = int64IdentitySchema()
}

func (r *AvailablePrefixResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
	r.cf = configureCache(req)
}

func (r *AvailablePrefixResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AvailablePrefixModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiPath := fmt.Sprintf("/api/ipam/prefixes/%d/available-prefixes/", plan.ParentPrefixId.ValueInt64())
	b := body{"prefix_length": plan.PrefixLength.ValueInt64()}
	availablePrefixBody(ctx, b, &plan, r.cf, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "allocating prefix", map[string]any{"path": apiPath, "prefix_length": plan.PrefixLength.ValueInt64()})
	obj, err := allocate[netbox.Prefix](ctx, r.client, apiPath, b)
	if err != nil {
		resp.Diagnostics.AddError("Error allocating netbox_available_prefix", err.Error())
		return
	}
	state := plan
	availablePrefixFromAPI(ctx, obj, &plan, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	provider.SetIdentityID(ctx, resp.Identity, state.Id, &resp.Diagnostics)
}

func (r *AvailablePrefixResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AvailablePrefixModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	obj, res, err := r.client.IpamAPI.IpamPrefixesRetrieve(ctx, id).Execute()
	if err != nil {
		werr := netbox.WrapError(err, res)
		if netbox.IsNotFound(werr) {
			tflog.Info(ctx, "prefix no longer exists, removing from state", map[string]any{"id": id})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading netbox_available_prefix", werr.Error())
		return
	}
	prior := state
	availablePrefixFromAPI(ctx, obj, &prior, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	provider.SetIdentityID(ctx, resp.Identity, state.Id, &resp.Diagnostics)
}

func (r *AvailablePrefixResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AvailablePrefixModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	patch := netbox.NewPatchedWritablePrefixRequest()
	if plan.ScopeType.IsNull() {
		patch.SetScopeTypeNil()
	} else if !plan.ScopeType.IsUnknown() {
		patch.SetScopeType(plan.ScopeType.ValueString())
	}
	if plan.ScopeId.IsNull() {
		patch.SetScopeIdNil()
	} else if !plan.ScopeId.IsUnknown() {
		patch.SetScopeId(conv.Int32(plan.ScopeId))
	}
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
	if conv.Known(plan.IsPool) {
		patch.SetIsPool(plan.IsPool.ValueBool())
	}
	if conv.Known(plan.MarkUtilized) {
		patch.SetMarkUtilized(plan.MarkUtilized.ValueBool())
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
		patch.SetCustomFields(customfields.ToAPI(ctx, plan.CustomFields, r.cf, "ipam.prefix", &resp.Diagnostics))
	}
	if resp.Diagnostics.HasError() {
		return
	}
	obj, res, err := r.client.IpamAPI.IpamPrefixesPartialUpdate(ctx, id).PatchedWritablePrefixRequest(*patch).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating netbox_available_prefix", netbox.WrapError(err, res).Error())
		return
	}
	out := plan
	out.ParentPrefixId, out.PrefixLength = state.ParentPrefixId, state.PrefixLength
	availablePrefixFromAPI(ctx, obj, &plan, &out, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
	provider.SetIdentityID(ctx, resp.Identity, out.Id, &resp.Diagnostics)
}

func (r *AvailablePrefixResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AvailablePrefixModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	res, err := r.client.IpamAPI.IpamPrefixesDestroy(ctx, id).Execute()
	if err != nil {
		if werr := netbox.WrapError(err, res); !netbox.IsNotFound(werr) {
			resp.Diagnostics.AddError("Error deleting netbox_available_prefix", werr.Error())
		}
	}
}

func (r *AvailablePrefixResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	provider.ImportInt64ID(ctx, req, resp)
}

// availablePrefixBody fills the allocation request from the plan. The prefix
// itself and the VRF are chosen by NetBox.
func availablePrefixBody(ctx context.Context, b body, plan *AvailablePrefixModel, cf *customfields.Cache, diags *diag.Diagnostics) {
	b.nullableStr("scope_type", plan.ScopeType)
	b.nullableInt("scope_id", plan.ScopeId)
	b.nullableInt("tenant", plan.TenantId)
	b.str("status", plan.Status)
	b.nullableInt("role", plan.RoleId)
	b.boolean("is_pool", plan.IsPool)
	b.boolean("mark_utilized", plan.MarkUtilized)
	b.str("description", plan.Description)
	b.str("comments", plan.Comments)
	b.tags(ctx, plan.Tags, diags)
	b.customFields(ctx, plan.CustomFields, cf, "ipam.prefix", diags)
}

// availablePrefixFromAPI copies the API object into out.
func availablePrefixFromAPI(ctx context.Context, obj *netbox.Prefix, prior, out *AvailablePrefixModel, diags *diag.Diagnostics) {
	out.Id = types.Int64Value(int64(obj.GetId()))
	out.Prefix = conv.String(obj.GetPrefixOk())
	out.VrfId = conv.BriefID(obj.GetVrfOk())
	out.ScopeType = conv.String(obj.GetScopeTypeOk())
	out.ScopeId = conv.Int64From32(obj.GetScopeIdOk())
	out.TenantId = conv.BriefID(obj.GetTenantOk())
	out.Status = conv.Choice(obj.GetStatusOk())
	out.RoleId = conv.BriefID(obj.GetRoleOk())
	out.IsPool = conv.Bool(obj.GetIsPoolOk())
	out.MarkUtilized = conv.Bool(obj.GetMarkUtilizedOk())
	out.Description = conv.StringOrEmpty(obj.GetDescriptionOk())
	out.Comments = conv.StringOrEmpty(obj.GetCommentsOk())
	out.Tags = conv.TagsFromAPI(obj.GetTags())
	out.CustomFields = customfields.FromAPI(ctx, obj.GetCustomFields(), prior.CustomFields, diags)
	out.Url = conv.String(obj.GetUrlOk())
	out.Display = conv.String(obj.GetDisplayOk())
}
