// Package manual holds hand-written Terraform resources that the code
// generator cannot express: allocation endpoints (available IPs, prefixes,
// VLANs and ASNs) and the device / virtual machine primary IP assignment.
//
// Every resource registers itself with provider.RegisterResource from init()
// and is pulled into the binary by a blank import in main.go.
package manual

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/elliot/terraform-provider-netbox/internal/conv"
	"github.com/elliot/terraform-provider-netbox/internal/provider"
	"github.com/elliot/terraform-provider-netbox/netbox"
)

// configureClient extracts the generated API client from the provider data.
// It returns nil (without a diagnostic) while the provider is not configured
// yet, which the framework does during validation.
func configureClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *netbox.APIClient {
	if req.ProviderData == nil {
		return nil
	}
	pd, ok := req.ProviderData.(*provider.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *provider.ProviderData, got %T", req.ProviderData))
		return nil
	}
	return pd.API
}

// int64IdentitySchema is the identity schema shared by every resource here:
// the numeric NetBox ID.
func int64IdentitySchema() identityschema.Schema {
	return identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.Int64Attribute{RequiredForImport: true, Description: "NetBox object ID."},
		},
	}
}

// ---- shared schema attributes ----

func idAttribute(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{
		MarkdownDescription: desc,
		Computed:            true,
		PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
	}
}

func computedStringAttribute(desc string, stable bool) schema.StringAttribute {
	a := schema.StringAttribute{MarkdownDescription: desc, Computed: true}
	if stable {
		a.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	}
	return a
}

func computedInt64Attribute(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{
		MarkdownDescription: desc,
		Computed:            true,
		PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
	}
}

// choiceAttribute is an Optional+Computed string restricted to the given values
// whose server default is kept when omitted.
func choiceAttribute(desc string, values []string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		Validators:          []validator.String{stringvalidator.OneOf(values...)},
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

// nullableChoiceAttribute is an Optional string restricted to the given values
// that is sent as JSON null when omitted.
func nullableChoiceAttribute(desc string, values []string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Validators:          []validator.String{stringvalidator.OneOf(values...)},
	}
}

// blankStringAttribute is an Optional+Computed string defaulting to "".
func blankStringAttribute(desc string, maxLen int, extra ...validator.String) schema.StringAttribute {
	a := schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
	}
	if maxLen > 0 {
		a.Validators = append(a.Validators, stringvalidator.LengthAtMost(maxLen))
	}
	a.Validators = append(a.Validators, extra...)
	return a
}

func fkAttribute(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{MarkdownDescription: desc, Optional: true}
}

func tagsAttribute() schema.SetAttribute {
	return schema.SetAttribute{
		MarkdownDescription: "Slugs of the tags assigned to this object. Defaults to no tags when omitted.",
		ElementType:         types.StringType,
		Optional:            true,
		Computed:            true,
		Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
	}
}

func customFieldsAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: "Custom field values as a JSON object (`jsonencode({...})`). Only keys present in the configuration are tracked.",
		CustomType:          jsontypes.NormalizedType{},
		Optional:            true,
	}
}

// ---- raw request body helpers ----
//
// Allocation endpoints are called with a map[string]any body so that null and
// omitted values are explicit: a null Terraform value becomes JSON null, an
// unknown value is omitted (NetBox applies its default).

type body map[string]any

// str sets key to the string when known; unknown values are omitted.
func (b body) str(key string, v types.String) {
	if conv.Known(v) {
		b[key] = v.ValueString()
	}
}

// nullableStr sets key to the string, or to null when the value is null.
func (b body) nullableStr(key string, v types.String) {
	switch {
	case v.IsNull():
		b[key] = nil
	case !v.IsUnknown():
		b[key] = v.ValueString()
	}
}

// nullableInt sets key to the integer, or to null when the value is null.
func (b body) nullableInt(key string, v types.Int64) {
	switch {
	case v.IsNull():
		b[key] = nil
	case !v.IsUnknown():
		b[key] = v.ValueInt64()
	}
}

// boolean sets key when the value is known.
func (b body) boolean(key string, v types.Bool) {
	if conv.Known(v) {
		b[key] = v.ValueBool()
	}
}

// tags sets "tags" to the nested tag requests when known.
func (b body) tags(ctx context.Context, v types.Set, diags *diag.Diagnostics) {
	if conv.Known(v) {
		b["tags"] = conv.TagsToAPI(ctx, v, diags)
	}
}

// customFields sets "custom_fields" when configured.
func (b body) customFields(v jsontypes.Normalized, diags *diag.Diagnostics) {
	if conv.Known(v) {
		b["custom_fields"] = conv.JSONObjectToAPI(v, diags)
	}
}

// allocate POSTs a single-object allocation request to apiPath and returns the
// one object NetBox created. Allocation endpoints accept and return lists.
func allocate[T any](ctx context.Context, client *netbox.APIClient, apiPath string, b body) (*T, error) {
	var out []T
	if err := client.PostRaw(ctx, apiPath, []body{b}, &out); err != nil {
		return nil, err
	}
	if len(out) != 1 {
		return nil, fmt.Errorf("expected exactly one object from %s, got %d", apiPath, len(out))
	}
	return &out[0], nil
}

// int32ID validates a state ID for the generated client.
func int32ID(v types.Int64, diags *diag.Diagnostics) (int32, bool) {
	id, err := netbox.Int32ID(v.ValueInt64())
	if err != nil {
		diags.AddError("Invalid ID", err.Error())
		return 0, false
	}
	return id, true
}
