package manual

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/elliot/terraform-provider-netbox/internal/conv"
	"github.com/elliot/terraform-provider-netbox/internal/customfields"
	"github.com/elliot/terraform-provider-netbox/internal/provider"
	"github.com/elliot/terraform-provider-netbox/internal/provider/gen/objecttypes"
	"github.com/elliot/terraform-provider-netbox/netbox"
)

func init() { provider.RegisterResource(NewCustomFieldValueResource) }

var (
	_ resource.Resource                = &CustomFieldValueResource{}
	_ resource.ResourceWithConfigure   = &CustomFieldValueResource{}
	_ resource.ResourceWithImportState = &CustomFieldValueResource{}
)

// CustomFieldValueResource sets one custom field on an object that Terraform
// does not otherwise manage (or that another configuration manages).
type CustomFieldValueResource struct {
	client *netbox.APIClient
	cf     *customfields.Cache
}

// CustomFieldValueModel is the state of netbox_custom_field_value.
type CustomFieldValueModel struct {
	Id         types.String  `tfsdk:"id"`
	ObjectType types.String  `tfsdk:"object_type"`
	ObjectId   types.Int64   `tfsdk:"object_id"`
	Name       types.String  `tfsdk:"name"`
	Value      types.Dynamic `tfsdk:"value"`
}

// NewCustomFieldValueResource returns the resource.
func NewCustomFieldValueResource() resource.Resource { return &CustomFieldValueResource{} }

func (r *CustomFieldValueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_field_value"
}

func (r *CustomFieldValueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Sets a single custom field value on any NetBox object, identified by content type and ID. " +
			"Use it to annotate objects that are not managed by this Terraform configuration; for managed objects " +
			"prefer the `custom_fields` attribute of the resource itself. Destroying the resource clears the field (sets it to null).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic ID `<object_type>/<object_id>/<name>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"object_type": schema.StringAttribute{
				MarkdownDescription: "Content type of the object, e.g. `dcim.device`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.RegexMatches(objectTypeRe, "must look like app.model")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"object_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the object.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Custom field name.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": schema.DynamicAttribute{
				MarkdownDescription: "The value: a string, number, boolean, list or object matching the custom field type " +
					"(object fields take the related object ID, selection fields the choice value).",
				Required: true,
			},
		},
	}
}

func (r *CustomFieldValueResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
	r.cf = configureCache(req)
}

// objectPath returns the API detail path for the object.
func (r *CustomFieldValueResource) objectPath(m *CustomFieldValueModel, diags *diag.Diagnostics) string {
	base, ok := objecttypes.Paths[m.ObjectType.ValueString()]
	if !ok {
		diags.AddAttributeError(path.Root("object_type"), "Unknown object type",
			fmt.Sprintf("%q is not a NetBox object type with custom fields support known to this provider.", m.ObjectType.ValueString()))
		return ""
	}
	return fmt.Sprintf("%s%d/", base, m.ObjectId.ValueInt64())
}

// write PATCHes the single field and refreshes the model from the response.
func (r *CustomFieldValueResource) write(ctx context.Context, m *CustomFieldValueModel, remove bool, diags *diag.Diagnostics) {
	p := r.objectPath(m, diags)
	if diags.HasError() {
		return
	}
	var value any
	if !remove {
		wrapped := customfields.ToAPI(ctx, wrapField(m), r.cf, m.ObjectType.ValueString(), diags)
		if diags.HasError() {
			return
		}
		value = wrapped[m.Name.ValueString()]
	}
	var out struct {
		CustomFields map[string]any `json:"custom_fields"`
	}
	if err := r.client.PatchRaw(ctx, p, map[string]any{"custom_fields": map[string]any{m.Name.ValueString(): value}}, &out); err != nil {
		diags.AddError("Error setting custom field", err.Error())
		return
	}
	if !remove {
		r.fill(ctx, m, out.CustomFields, diags)
	}
}

// wrapField builds a one-key dynamic object {name: value} so the shared
// conversion helpers can be reused.
func wrapField(m *CustomFieldValueModel) types.Dynamic {
	obj, _ := types.ObjectValue(
		map[string]attrType{m.Name.ValueString(): m.Value.UnderlyingValue().Type(context.Background())},
		map[string]attrValue{m.Name.ValueString(): m.Value.UnderlyingValue()},
	)
	return types.DynamicValue(obj)
}

// fill copies the field value from the API map into the model, shaped after
// the configured value.
func (r *CustomFieldValueResource) fill(ctx context.Context, m *CustomFieldValueModel, api map[string]any, diags *diag.Diagnostics) {
	shaped := customfields.FromAPI(ctx, api, wrapField(m), diags)
	if obj, ok := shaped.UnderlyingValue().(types.Object); ok {
		if v, ok := obj.Attributes()[m.Name.ValueString()]; ok {
			m.Value = types.DynamicValue(v)
		}
	}
	m.Id = types.StringValue(fmt.Sprintf("%s/%d/%s", m.ObjectType.ValueString(), m.ObjectId.ValueInt64(), m.Name.ValueString()))
}

func (r *CustomFieldValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CustomFieldValueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.write(ctx, &plan, false, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CustomFieldValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CustomFieldValueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p := r.objectPath(&state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	var out struct {
		CustomFields map[string]any `json:"custom_fields"`
	}
	if err := r.client.GetRaw(ctx, p, &out); err != nil {
		if netbox.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading custom field", err.Error())
		return
	}
	if v, ok := out.CustomFields[state.Name.ValueString()]; !ok || v == nil {
		// Field cleared or removed outside Terraform.
		resp.State.RemoveResource(ctx)
		return
	}
	if !conv.Known(state.Value) {
		// Imported: infer the value type from the API.
		state.Value = customfields.Infer(map[string]any{state.Name.ValueString(): out.CustomFields[state.Name.ValueString()]})
		if obj, ok := state.Value.UnderlyingValue().(types.Object); ok {
			state.Value = types.DynamicValue(obj.Attributes()[state.Name.ValueString()])
		}
		state.Id = types.StringValue(fmt.Sprintf("%s/%d/%s", state.ObjectType.ValueString(), state.ObjectId.ValueInt64(), state.Name.ValueString()))
	} else {
		r.fill(ctx, &state, out.CustomFields, &resp.Diagnostics)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CustomFieldValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CustomFieldValueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.write(ctx, &plan, false, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CustomFieldValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CustomFieldValueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var d diag.Diagnostics
	r.write(ctx, &state, true, &d)
	for _, e := range d.Errors() {
		if !strings.Contains(e.Detail(), "404") {
			resp.Diagnostics.Append(e)
		}
	}
}

// ImportState accepts "<object_type>/<object_id>/<name>".
func (r *CustomFieldValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected <object_type>/<object_id>/<field name>, e.g. dcim.device/12/cost_center.")
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "The object ID must be numeric.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_type"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value"), types.DynamicNull())...)
}
