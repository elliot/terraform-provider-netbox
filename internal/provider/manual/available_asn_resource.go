package manual

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/elliot/terraform-provider-netbox/internal/conv"
	"github.com/elliot/terraform-provider-netbox/internal/provider"
	"github.com/elliot/terraform-provider-netbox/netbox"
)

func init() { provider.RegisterResource(NewAvailableAsnResource) }

// AvailableAsnModel is the Terraform state of netbox_available_asn.
type AvailableAsnModel struct {
	Id           types.Int64          `tfsdk:"id"`
	AsnRangeId   types.Int64          `tfsdk:"asn_range_id"`
	Asn          types.Int64          `tfsdk:"asn"`
	RirId        types.Int64          `tfsdk:"rir_id"`
	TenantId     types.Int64          `tfsdk:"tenant_id"`
	Description  types.String         `tfsdk:"description"`
	Comments     types.String         `tfsdk:"comments"`
	Tags         types.Set            `tfsdk:"tags"`
	CustomFields jsontypes.Normalized `tfsdk:"custom_fields"`
	Url          types.String         `tfsdk:"url"`
	Display      types.String         `tfsdk:"display"`
}

var (
	_ resource.Resource                = &AvailableAsnResource{}
	_ resource.ResourceWithConfigure   = &AvailableAsnResource{}
	_ resource.ResourceWithImportState = &AvailableAsnResource{}
	_ resource.ResourceWithIdentity    = &AvailableAsnResource{}
)

// AvailableAsnResource allocates the next free ASN of an ASN range and then
// manages it like a netbox_asn.
type AvailableAsnResource struct {
	client *netbox.APIClient
}

// NewAvailableAsnResource returns a new netbox_available_asn resource.
func NewAvailableAsnResource() resource.Resource { return &AvailableAsnResource{} }

func (r *AvailableAsnResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_available_asn"
}

func (r *AvailableAsnResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Allocates the next available ASN in an ASN range (`POST /api/ipam/asn-ranges/{id}/available-asns/`) " +
			"and manages the resulting ASN (`/api/ipam/asns/`). The RIR is inherited from the range. " +
			"Changing `asn_range_id` allocates a new ASN.\n\n" +
			"After `terraform import` the range is unknown, so `asn_range_id` stays null in state; " +
			"setting it in the configuration of an imported resource forces a replacement.",
		Attributes: map[string]schema.Attribute{
			"id": idAttribute("The numeric ID of the ASN object in NetBox (not the AS number itself)."),
			"asn_range_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the ASN range (`netbox_asn_range`) to allocate from.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"asn":           computedInt64Attribute("The allocated 16- or 32-bit autonomous system number."),
			"rir_id":        computedInt64Attribute("ID of the RIR (`netbox_rir`), inherited from the ASN range."),
			"tenant_id":     fkAttribute("ID of the Tenant (`netbox_tenant`)."),
			"description":   blankStringAttribute("Description.", 200),
			"comments":      blankStringAttribute("Comments.", 0),
			"tags":          tagsAttribute(),
			"custom_fields": customFieldsAttribute(),
			"url":           computedStringAttribute("URL of the object in the NetBox API.", true),
			"display":       computedStringAttribute("Display name of the object.", false),
		},
	}
}

func (r *AvailableAsnResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = int64IdentitySchema()
}

func (r *AvailableAsnResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *AvailableAsnResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AvailableAsnModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiPath := fmt.Sprintf("/api/ipam/asn-ranges/%d/available-asns/", plan.AsnRangeId.ValueInt64())
	b := body{}
	availableAsnBody(ctx, b, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "allocating ASN", map[string]any{"path": apiPath})
	obj, err := allocate[netbox.ASN](ctx, r.client, apiPath, b)
	if err != nil {
		resp.Diagnostics.AddError("Error allocating netbox_available_asn", err.Error())
		return
	}
	state := plan
	availableAsnFromAPI(obj, &plan, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	provider.SetIdentityID(ctx, resp.Identity, state.Id, &resp.Diagnostics)
}

func (r *AvailableAsnResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AvailableAsnModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	obj, res, err := r.client.IpamAPI.IpamAsnsRetrieve(ctx, id).Execute()
	if err != nil {
		werr := netbox.WrapError(err, res)
		if netbox.IsNotFound(werr) {
			tflog.Info(ctx, "ASN no longer exists, removing from state", map[string]any{"id": id})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading netbox_available_asn", werr.Error())
		return
	}
	prior := state
	availableAsnFromAPI(obj, &prior, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	provider.SetIdentityID(ctx, resp.Identity, state.Id, &resp.Diagnostics)
}

func (r *AvailableAsnResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AvailableAsnModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	patch := netbox.NewPatchedASNRequest()
	if plan.TenantId.IsNull() {
		patch.SetTenantNil()
	} else if !plan.TenantId.IsUnknown() {
		patch.SetTenant(conv.Int32(plan.TenantId))
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
		patch.SetCustomFields(conv.JSONObjectToAPI(plan.CustomFields, &resp.Diagnostics))
	}
	if resp.Diagnostics.HasError() {
		return
	}
	obj, res, err := r.client.IpamAPI.IpamAsnsPartialUpdate(ctx, id).PatchedASNRequest(*patch).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating netbox_available_asn", netbox.WrapError(err, res).Error())
		return
	}
	out := plan
	out.AsnRangeId = state.AsnRangeId
	availableAsnFromAPI(obj, &plan, &out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
	provider.SetIdentityID(ctx, resp.Identity, out.Id, &resp.Diagnostics)
}

func (r *AvailableAsnResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AvailableAsnModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := int32ID(state.Id, &resp.Diagnostics)
	if !ok {
		return
	}
	res, err := r.client.IpamAPI.IpamAsnsDestroy(ctx, id).Execute()
	if err != nil {
		if werr := netbox.WrapError(err, res); !netbox.IsNotFound(werr) {
			resp.Diagnostics.AddError("Error deleting netbox_available_asn", werr.Error())
		}
	}
}

func (r *AvailableAsnResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	provider.ImportInt64ID(ctx, req, resp)
}

// availableAsnBody fills the allocation request from the plan. The AS number
// and RIR are chosen by NetBox.
func availableAsnBody(ctx context.Context, b body, plan *AvailableAsnModel, diags *diag.Diagnostics) {
	b.nullableInt("tenant", plan.TenantId)
	b.str("description", plan.Description)
	b.str("comments", plan.Comments)
	b.tags(ctx, plan.Tags, diags)
	b.customFields(plan.CustomFields, diags)
}

// availableAsnFromAPI copies the API object into out.
func availableAsnFromAPI(obj *netbox.ASN, prior, out *AvailableAsnModel) {
	out.Id = types.Int64Value(int64(obj.GetId()))
	out.Asn = conv.Int64From64(obj.GetAsnOk())
	out.RirId = conv.BriefID(obj.GetRirOk())
	out.TenantId = conv.BriefID(obj.GetTenantOk())
	out.Description = conv.StringOrEmpty(obj.GetDescriptionOk())
	out.Comments = conv.StringOrEmpty(obj.GetCommentsOk())
	out.Tags = conv.TagsFromAPI(obj.GetTags())
	out.CustomFields = conv.CustomFieldsFromAPI(obj.GetCustomFields(), prior.CustomFields)
	out.Url = conv.String(obj.GetUrlOk())
	out.Display = conv.String(obj.GetDisplayOk())
}
