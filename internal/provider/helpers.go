package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ImportInt64ID implements ImportState for resources identified by a numeric
// NetBox ID. It accepts `terraform import netbox_x.y 123` as well as identity
// based imports (`identity = { id = 123 }`).
func ImportInt64ID(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var id int64
	switch {
	case req.ID != "":
		n, err := strconv.ParseInt(req.ID, 10, 64)
		if err != nil || n <= 0 {
			resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected the numeric NetBox object ID, got %q.", req.ID))
			return
		}
		id = n
	case req.Identity != nil:
		var v types.Int64
		resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("id"), &v)...)
		if resp.Diagnostics.HasError() {
			return
		}
		id = v.ValueInt64()
	default:
		resp.Diagnostics.AddError("Invalid import", "An import ID or identity is required.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.Int64Value(id))...)
	SetIdentityID(ctx, resp.Identity, types.Int64Value(id), &resp.Diagnostics)
}

// SetIdentityID stores the numeric ID in the resource identity when the
// request carries one.
func SetIdentityID(ctx context.Context, identity *tfsdk.ResourceIdentity, id types.Int64, diags *diag.Diagnostics) {
	if identity == nil || identity.Raw.IsNull() && identity.Schema == nil {
		return
	}
	diags.Append(identity.SetAttribute(ctx, path.Root("id"), id)...)
}

// AttrTypes returns the attribute types of a data source attribute map, used
// to build nested object types for list data sources.
func AttrTypes(attrs map[string]dsschema.Attribute) map[string]attr.Type {
	out := make(map[string]attr.Type, len(attrs))
	for k, a := range attrs {
		out[k] = a.GetType()
	}
	return out
}
