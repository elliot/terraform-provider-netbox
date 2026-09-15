package render

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/elliot/terraform-provider-netbox/internal/gen/model"
	"github.com/elliot/terraform-provider-netbox/internal/gen/naming"
)

// ---- small helpers ----

func q(s string) string { return strconv.Quote(s) }

// field is the Go struct field name of an attribute.
func field(a model.Attr) string { return naming.GoIdent(a.Name) }

// itemType is the Go type name for a nested list item model.
func itemType(r *model.Resource, a model.Attr) string {
	return r.GoName + naming.GoIdent(a.Name) + "Item"
}

// tfType returns the framework value type of an attribute's model field.
func tfType(a model.Attr) string {
	switch a.Kind {
	case model.KindString, model.KindChoice:
		return "types.String"
	case model.KindCustomFieldsJSON:
		return "jsontypes.Normalized"
	case model.KindInt, model.KindChoiceInt, model.KindFK:
		return "types.Int64"
	case model.KindFloat:
		return "types.Float64"
	case model.KindBool:
		return "types.Bool"
	case model.KindDateTime:
		return "timetypes.RFC3339"
	case model.KindCustomFields:
		return "types.Dynamic"
	case model.KindJSON:
		return "jsontypes.Normalized"
	case model.KindFKList, model.KindIntList, model.KindTags:
		if a.OrderedList {
			return "types.List"
		}
		return "types.Set"
	case model.KindStringList:
		if a.OrderedList {
			return "types.List"
		}
		return "types.Set"
	case model.KindNestedList, model.KindIntRangeList, model.KindAnyList:
		return "types.List"
	}
	return "types.String"
}

// attrType returns the attr.Type expression of an attribute (for object types).
func attrType(r *model.Resource, a model.Attr) string {
	switch a.Kind {
	case model.KindString, model.KindChoice:
		return "types.StringType"
	case model.KindInt, model.KindChoiceInt, model.KindFK:
		return "types.Int64Type"
	case model.KindFloat:
		return "types.Float64Type"
	case model.KindBool:
		return "types.BoolType"
	case model.KindDateTime:
		return "timetypes.RFC3339Type{}"
	case model.KindCustomFields:
		return "types.DynamicType"
	case model.KindJSON, model.KindCustomFieldsJSON:
		return "jsontypes.NormalizedType{}"
	case model.KindFKList, model.KindIntList:
		if a.OrderedList {
			return "types.ListType{ElemType: types.Int64Type}"
		}
		return "types.SetType{ElemType: types.Int64Type}"
	case model.KindStringList, model.KindTags:
		if a.OrderedList {
			return "types.ListType{ElemType: types.StringType}"
		}
		return "types.SetType{ElemType: types.StringType}"
	case model.KindIntRangeList:
		return "types.ListType{ElemType: types.ListType{ElemType: types.Int64Type}}"
	case model.KindAnyList:
		return "types.ListType{ElemType: types.ListType{ElemType: types.StringType}}"
	case model.KindNestedList:
		return fmt.Sprintf("types.ListType{ElemType: types.ObjectType{AttrTypes: %sAttrTypes}}", lower(itemType(r, a)))
	}
	return "types.StringType"
}

func lower(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// description builds the markdown description of an attribute.
func description(a model.Attr) string {
	d := strings.TrimSpace(a.Description)
	// NetBox puts the choice list into the description as "* `x` - Label" lines; keep the first sentence.
	if i := strings.Index(d, "\n\n*"); i >= 0 {
		d = strings.TrimSpace(d[:i])
	} else if strings.HasPrefix(d, "*") {
		d = ""
	}
	if d != "" && !strings.HasSuffix(d, ".") {
		d += "."
	}
	if d == "" {
		d = naming.Title(a.JSON) + "."
	}
	switch {
	case len(a.Enum) > 0 && len(a.Enum) <= 40:
		d += " Valid values: `" + strings.Join(a.Enum, "`, `") + "`."
	case len(a.Enum) > 40:
		d += fmt.Sprintf(" One of %d valid NetBox choices (see the NetBox documentation).", len(a.Enum))
	case len(a.EnumInts) > 0:
		parts := make([]string, 0, len(a.EnumInts))
		for _, n := range a.EnumInts {
			parts = append(parts, strconv.FormatInt(n, 10))
		}
		d += " Valid values: `" + strings.Join(parts, "`, `") + "`."
	}
	if a.Kind == model.KindFK && a.Target != "" && !strings.Contains(d, "netbox_") {
		d += fmt.Sprintf(" References `netbox_%s`.", a.Target)
	}
	if a.Kind == model.KindFKList && a.Target != "" && !strings.Contains(d, "netbox_") {
		d += fmt.Sprintf(" References `netbox_%s`.", a.Target)
	}
	switch {
	case a.DefaultEmptySet:
		d += " Defaults to an empty set."
	case a.DefaultEmptyString:
		d += " Defaults to an empty string."
	case a.Computed && !a.ReadOnly && !a.Required:
		d += " Defaults to the NetBox server default when omitted."
	}
	if a.WriteOnly {
		d += " Write-only: NetBox does not return this value."
	}
	return d
}

var goRegexpOK = func(p string) bool { _, err := regexp.Compile(p); return err == nil }

// enumVar is the package-level variable holding choice values.
func enumVar(r *model.Resource, a model.Attr, parent string) string {
	return lower(r.GoName) + parent + naming.GoIdent(a.Name) + "Values"
}

// ---- schema emission ----

// schemaAttr renders one attribute of a resource schema (pkg "schema") or a
// data source schema (pkg "dsschema", everything computed except lookups).
func schemaAttr(r *model.Resource, a model.Attr, pkg string, dataSource bool, lookup bool, parent string) string {
	var b strings.Builder
	typ := ""
	switch a.Kind {
	case model.KindString, model.KindChoice:
		typ = "StringAttribute"
	case model.KindInt, model.KindChoiceInt, model.KindFK:
		typ = "Int64Attribute"
	case model.KindFloat:
		typ = "Float64Attribute"
	case model.KindBool:
		typ = "BoolAttribute"
	case model.KindDateTime:
		typ = "StringAttribute"
	case model.KindCustomFields:
		typ = "DynamicAttribute"
	case model.KindJSON, model.KindCustomFieldsJSON:
		typ = "StringAttribute"
	case model.KindFKList, model.KindIntList, model.KindStringList, model.KindTags:
		if a.OrderedList {
			typ = "ListAttribute"
		} else {
			typ = "SetAttribute"
		}
	case model.KindIntRangeList, model.KindAnyList:
		typ = "ListAttribute"
	case model.KindNestedList:
		typ = "ListNestedAttribute"
	}
	fmt.Fprintf(&b, "%q: %s.%s{\n", a.Name, pkg, typ)
	fmt.Fprintf(&b, "MarkdownDescription: %s,\n", q(description(a)))
	switch a.Kind {
	case model.KindDateTime:
		b.WriteString("CustomType: timetypes.RFC3339Type{},\n")
	case model.KindJSON, model.KindCustomFieldsJSON:
		b.WriteString("CustomType: jsontypes.NormalizedType{},\n")
	case model.KindFKList, model.KindIntList:
		b.WriteString("ElementType: types.Int64Type,\n")
	case model.KindStringList, model.KindTags:
		b.WriteString("ElementType: types.StringType,\n")
	case model.KindIntRangeList:
		b.WriteString("ElementType: types.ListType{ElemType: types.Int64Type},\n")
	case model.KindAnyList:
		b.WriteString("ElementType: types.ListType{ElemType: types.StringType},\n")
	case model.KindNestedList:
		fmt.Fprintf(&b, "NestedObject: %s.NestedAttributeObject{Attributes: map[string]%s.Attribute{\n", pkg, pkg)
		for _, n := range a.Nested {
			b.WriteString(schemaAttr(r, n, pkg, dataSource, false, naming.GoIdent(a.Name)))
		}
		b.WriteString("}},\n")
	}
	if a.Sensitive {
		b.WriteString("Sensitive: true,\n")
	}

	if dataSource {
		if lookup {
			b.WriteString("Optional: true,\nComputed: true,\n")
		} else {
			b.WriteString("Computed: true,\n")
		}
		b.WriteString("},\n")
		return b.String()
	}

	switch {
	case a.ReadOnly:
		b.WriteString("Computed: true,\n")
	case a.Required:
		b.WriteString("Required: true,\n")
	default:
		b.WriteString("Optional: true,\n")
		if a.Computed {
			b.WriteString("Computed: true,\n")
		}
	}

	// Validators.
	var vals []string
	vtype := ""
	switch a.Kind {
	case model.KindString:
		vtype = "String"
		if a.MaxLength != nil && *a.MaxLength > 0 {
			vals = append(vals, fmt.Sprintf("stringvalidator.LengthAtMost(%d)", *a.MaxLength))
		}
		if a.Pattern != "" && goRegexpOK(a.Pattern) && !a.Nullable {
			vals = append(vals, fmt.Sprintf("stringvalidator.RegexMatches(regexp.MustCompile(%s), %q)", q(a.Pattern), "must match "+a.Pattern))
		}
	case model.KindChoice:
		vtype = "String"
		if len(a.Enum) > 0 {
			vals = append(vals, fmt.Sprintf("stringvalidator.OneOf(%s...)", enumVar(r, a, parent)))
		}
	case model.KindChoiceInt:
		vtype = "Int64"
		if len(a.EnumInts) > 0 {
			vals = append(vals, fmt.Sprintf("int64validator.OneOf(%s...)", enumVar(r, a, parent)))
		}
	case model.KindStringList:
		if len(a.Enum) > 0 {
			if a.OrderedList {
				vtype = "List"
				vals = append(vals, fmt.Sprintf("listvalidator.ValueStringsAre(stringvalidator.OneOf(%s...))", enumVar(r, a, parent)))
			} else {
				vtype = "Set"
				vals = append(vals, fmt.Sprintf("setvalidator.ValueStringsAre(stringvalidator.OneOf(%s...))", enumVar(r, a, parent)))
			}
		}
	}
	if len(vals) > 0 {
		fmt.Fprintf(&b, "Validators: []validator.%s{%s},\n", vtype, strings.Join(vals, ", "))
	}

	// Defaults and plan modifiers (top-level attributes only).
	if parent == "" {
		pmType := planModifierType(a)
		var pms []string
		if a.ReadOnly {
			if a.Name == "id" || a.Name == "url" || a.Name == "created" {
				pms = append(pms, fmt.Sprintf("%splanmodifier.UseStateForUnknown()", strings.ToLower(pmType)))
			}
		} else if a.Computed {
			switch {
			case a.DefaultEmptyString:
				b.WriteString("Default: stringdefault.StaticString(\"\"),\n")
			case a.DefaultEmptySet && !a.OrderedList:
				elem := "types.StringType"
				if a.Kind == model.KindFKList || a.Kind == model.KindIntList {
					elem = "types.Int64Type"
				}
				fmt.Fprintf(&b, "Default: setdefault.StaticValue(types.SetValueMust(%s, []attr.Value{})),\n", elem)
			case a.DefaultEmptySet && a.OrderedList:
				elem := "types.StringType"
				if a.Kind == model.KindFKList || a.Kind == model.KindIntList {
					elem = "types.Int64Type"
				}
				fmt.Fprintf(&b, "Default: listdefault.StaticValue(types.ListValueMust(%s, []attr.Value{})),\n", elem)
			default:
				if pmType != "" {
					pms = append(pms, fmt.Sprintf("%splanmodifier.UseStateForUnknown()", strings.ToLower(pmType)))
				}
			}
		}
		if a.RequiresReplace && pmType != "" {
			pms = append(pms, fmt.Sprintf("%splanmodifier.RequiresReplace()", strings.ToLower(pmType)))
		}
		if len(pms) > 0 {
			fmt.Fprintf(&b, "PlanModifiers: []planmodifier.%s{%s},\n", pmType, strings.Join(pms, ", "))
		}
	}
	b.WriteString("},\n")
	return b.String()
}

func planModifierType(a model.Attr) string {
	switch a.Kind {
	case model.KindString, model.KindChoice, model.KindDateTime, model.KindJSON:
		return "String"
	case model.KindCustomFields:
		return "Dynamic"
	case model.KindInt, model.KindChoiceInt, model.KindFK:
		return "Int64"
	case model.KindFloat:
		return "Float64"
	case model.KindBool:
		return "Bool"
	case model.KindFKList, model.KindIntList, model.KindStringList, model.KindTags:
		if a.OrderedList {
			return "List"
		}
		return "Set"
	case model.KindNestedList, model.KindIntRangeList, model.KindAnyList:
		return "List"
	}
	return ""
}

// enumVars renders package-level slices for every choice attribute.
func enumVars(r *model.Resource) string {
	var b strings.Builder
	var walk func(attrs []model.Attr, parent string)
	walk = func(attrs []model.Attr, parent string) {
		for _, a := range attrs {
			switch {
			case len(a.Enum) > 0 && (a.Kind == model.KindChoice || a.Kind == model.KindStringList):
				fmt.Fprintf(&b, "// %s lists the valid values of %s.%s.\nvar %s = []string{", enumVar(r, a, parent), r.TFType(), a.Name, enumVar(r, a, parent))
				for i, e := range a.Enum {
					if i%8 == 0 {
						b.WriteString("\n")
					}
					b.WriteString(q(e) + ", ")
				}
				b.WriteString("\n}\n\n")
			case len(a.EnumInts) > 0 && a.Kind == model.KindChoiceInt:
				fmt.Fprintf(&b, "// %s lists the valid values of %s.%s.\nvar %s = []int64{", enumVar(r, a, parent), r.TFType(), a.Name, enumVar(r, a, parent))
				for _, e := range a.EnumInts {
					b.WriteString(strconv.FormatInt(e, 10) + ", ")
				}
				b.WriteString("}\n\n")
			}
			if len(a.Nested) > 0 {
				walk(a.Nested, naming.GoIdent(a.Name))
			}
		}
	}
	walk(r.Attrs, "")
	return b.String()
}

// ---- model emission ----

func modelStruct(name string, attrs []model.Attr) string {
	var b strings.Builder
	fmt.Fprintf(&b, "type %s struct {\n", name)
	for _, a := range attrs {
		fmt.Fprintf(&b, "%s %s `tfsdk:%q`\n", field(a), tfType(a), a.Name)
	}
	b.WriteString("}\n\n")
	return b.String()
}

// nestedTypes renders item model structs and attr type maps for nested lists.
func nestedTypes(r *model.Resource, attrs []model.Attr) string {
	var b strings.Builder
	for _, a := range attrs {
		if a.Kind != model.KindNestedList {
			continue
		}
		it := itemType(r, a)
		fmt.Fprintf(&b, "// %s is one element of %s.%s.\n", it, r.TFType(), a.Name)
		b.WriteString(modelStruct(it, a.Nested))
		fmt.Fprintf(&b, "var %sAttrTypes = map[string]attr.Type{\n", lower(it))
		for _, n := range a.Nested {
			fmt.Fprintf(&b, "%q: %s,\n", n.Name, attrType(r, n))
		}
		b.WriteString("}\n\n")
	}
	return b.String()
}

// ---- Terraform -> API ----

// scalarExpr returns the Go expression converting a known model field to the
// client's scalar type.
func scalarExpr(a model.Attr, v string) string {
	switch a.Kind {
	case model.KindString, model.KindChoice:
		return v + ".ValueString()"
	case model.KindInt, model.KindChoiceInt, model.KindFK:
		if a.Int64 {
			return v + ".ValueInt64()"
		}
		return "conv.Int32(" + v + ")"
	case model.KindFloat:
		if a.Float32 {
			return "conv.Float32(" + v + ")"
		}
		return v + ".ValueFloat64()"
	case model.KindBool:
		return v + ".ValueBool()"
	case model.KindDateTime:
		return "conv.Time(" + v + ", diags)"
	}
	return v
}

// collectionExpr returns the expression converting a collection field.
func collectionExpr(a model.Attr, v string) string {
	switch a.Kind {
	case model.KindFKList, model.KindIntList:
		if a.Int64 {
			return "conv.Int64s(ctx, " + v + ", diags)"
		}
		return "conv.Int32s(ctx, " + v + ", diags)"
	case model.KindStringList:
		return "conv.Strings(ctx, " + v + ", diags)"
	case model.KindTags:
		return "conv.TagsToAPI(ctx, " + v + ", diags)"
	case model.KindCustomFields:
		return "customfields.ToAPI(ctx, " + v + ", cf, objectType, diags)"
	case model.KindJSON:
		if a.StringMap {
			return "conv.JSONStringMapToAPI(" + v + ", diags)"
		}
		return "conv.JSONToAPI(" + v + ", diags)"
	case model.KindIntRangeList:
		return "conv.IntRangesToAPI(ctx, " + v + ", diags)"
	case model.KindAnyList:
		return "conv.AnyListToAPI(ctx, " + v + ", diags)"
	}
	return v
}

func isScalarKind(k model.Kind) bool {
	switch k {
	case model.KindString, model.KindChoice, model.KindInt, model.KindChoiceInt, model.KindFK, model.KindFloat, model.KindBool, model.KindDateTime:
		return true
	}
	return false
}

// nestedListBuild renders code building the client slice for a nested list
// into variable `dst` from model field `src`.
func nestedListBuild(r *model.Resource, a model.Attr, src, dst string) string {
	var b strings.Builder
	it := itemType(r, a)
	fmt.Fprintf(&b, "%s := []netbox.%s{}\n", dst, a.NestedReqType)
	fmt.Fprintf(&b, "if conv.Known(%s) {\n", src)
	fmt.Fprintf(&b, "var items []%s\n", it)
	fmt.Fprintf(&b, "diags.Append(%s.ElementsAs(ctx, &items, false)...)\n", src)
	b.WriteString("for _, it := range items {\n")
	// constructor
	var args []string
	reqSet := map[string]bool{}
	for _, rq := range a.NestedItemRequired {
		reqSet[rq] = true
		var na *model.Attr
		for i := range a.Nested {
			if a.Nested[i].JSON == rq {
				na = &a.Nested[i]
			}
		}
		if na == nil {
			args = append(args, "/* missing required "+rq+" */")
			continue
		}
		if isScalarKind(na.Kind) {
			if na.Nullable {
				args = append(args, "conv.NullableInt32(it."+field(*na)+")")
			} else {
				args = append(args, scalarExpr(*na, "it."+field(*na)))
			}
		} else {
			args = append(args, collectionExpr(*na, "it."+field(*na)))
		}
	}
	fmt.Fprintf(&b, "e := netbox.New%s(%s)\n", a.NestedReqType, strings.Join(args, ", "))
	for _, na := range a.Nested {
		if reqSet[na.JSON] {
			continue
		}
		b.WriteString(setterCode(na, "e", "it."+field(na), false))
	}
	fmt.Fprintf(&b, "%s = append(%s, *e)\n", dst, dst)
	b.WriteString("}\n}\n")
	return b.String()
}

// setterCode renders the statements applying model field v to request body
// `body` for an optional attribute (or a required one on PATCH).
func setterCode(a model.Attr, body, v string, patch bool) string {
	set := body + ".Set" + a.GoField
	switch {
	case a.ReadOnly:
		return ""
	case a.Kind == model.KindNestedList:
		return "" // handled by caller
	case isScalarKind(a.Kind):
		switch {
		case a.Required && a.Nullable:
			return fmt.Sprintf("if %s.IsNull() {\n%sNil()\n} else if !%s.IsUnknown() {\n%s(%s)\n}\n", v, set, v, set, scalarExpr(a, v))
		case a.Required:
			return fmt.Sprintf("if conv.Known(%s) {\n%s(%s)\n}\n", v, set, scalarExpr(a, v))
		case a.Nullable && !a.Computed:
			// Optional nullable: on PATCH (only emitted when the value changed) an
			// explicit null clears the field; on create a null is the default and
			// some NetBox serializers reject it, so it is omitted.
			if patch {
				return fmt.Sprintf("if %s.IsNull() {\n%sNil()\n} else if !%s.IsUnknown() {\n%s(%s)\n}\n", v, set, v, set, scalarExpr(a, v))
			}
			return fmt.Sprintf("if conv.Known(%s) {\n%s(%s)\n}\n", v, set, scalarExpr(a, v))
		case a.Nullable && a.Computed:
			// Optional+Computed nullable (choices, numbers, datetimes): a null plan
			// value only ever mirrors a null server value, and NetBox rejects an
			// explicit null for choice fields without allow_blank, so never send it.
			return fmt.Sprintf("if conv.Known(%s) {\n%s(%s)\n}\n", v, set, scalarExpr(a, v))
		case a.DefaultEmptyString:
			return fmt.Sprintf("if !%s.IsUnknown() {\n%s(%s)\n}\n", v, set, scalarExpr(a, v))
		default:
			// Non-nullable optional: cannot be cleared, send when known.
			return fmt.Sprintf("if conv.Known(%s) {\n%s(%s)\n}\n", v, set, scalarExpr(a, v))
		}
	case a.Kind == model.KindCustomFields || a.Kind == model.KindJSON:
		return fmt.Sprintf("if conv.Known(%s) {\n%s(%s)\n}\n", v, set, collectionExpr(a, v))
	default:
		return fmt.Sprintf("if conv.Known(%s) {\n%s(%s)\n}\n", v, set, collectionExpr(a, v))
	}
}

// ctorArgs renders the pre-statements and argument list for the create constructor.
func ctorArgs(r *model.Resource) (pre string, args []string, err error) {
	var b strings.Builder
	for _, rq := range r.CreateRequired {
		var a *model.Attr
		for i := range r.Attrs {
			if r.Attrs[i].JSON == rq && !r.Attrs[i].ReadOnly {
				a = &r.Attrs[i]
			}
		}
		if a == nil {
			return "", nil, fmt.Errorf("required property %q is not an attribute (skipped in overrides?)", rq)
		}
		v := "plan." + field(*a)
		switch {
		case a.Kind == model.KindNestedList:
			dst := lower(field(*a)) + "Items"
			b.WriteString(nestedListBuild(r, *a, v, dst))
			args = append(args, dst)
		case isScalarKind(a.Kind) && a.Nullable:
			args = append(args, "conv.NullableInt32("+v+")")
		case isScalarKind(a.Kind):
			args = append(args, scalarExpr(*a, v))
		default:
			args = append(args, collectionExpr(*a, v))
		}
	}
	return b.String(), args, nil
}

// toAPIFunc renders siteToCreate / siteToPatch.
func toAPIFunc(r *model.Resource, patch bool) (string, error) {
	var b strings.Builder
	name, typ := lower(r.GoName)+"ToCreate", r.CreateType
	if patch {
		name, typ = lower(r.GoName)+"ToPatch", r.PatchType
	}
	cfParams := ""
	if r.HasCustomFields {
		cfParams = ", cf *customfields.Cache"
	}
	if patch {
		fmt.Fprintf(&b, "// %s builds the %s request body with every attribute whose planned value differs from state.\n", name, typ)
		fmt.Fprintf(&b, "func %s(ctx context.Context, plan, state *%sModel%s, diags *diag.Diagnostics) *netbox.%s {\n", name, r.GoName, cfParams, typ)
	} else {
		fmt.Fprintf(&b, "// %s builds the %s request body from the plan.\n", name, typ)
		fmt.Fprintf(&b, "func %s(ctx context.Context, plan *%sModel%s, diags *diag.Diagnostics) *netbox.%s {\n", name, r.GoName, cfParams, typ)
	}
	if r.HasCustomFields {
		fmt.Fprintf(&b, "const objectType = %q\n", r.ObjectType())
	}
	required := map[string]bool{}
	if patch {
		fmt.Fprintf(&b, "body := netbox.New%s()\n", typ)
	} else {
		pre, args, err := ctorArgs(r)
		if err != nil {
			return "", err
		}
		b.WriteString(pre)
		fmt.Fprintf(&b, "body := netbox.New%s(%s)\n", typ, strings.Join(args, ", "))
		for _, rq := range r.CreateRequired {
			required[rq] = true
		}
	}
	for _, a := range r.Attrs {
		if a.ReadOnly || required[a.JSON] {
			continue
		}
		v := "plan." + field(a)
		var code string
		if a.Kind == model.KindNestedList {
			dst := lower(field(a)) + "Items"
			code = fmt.Sprintf("if conv.Known(%s) {\n%sbody.Set%s(%s)\n}\n", v, nestedListBuild(r, a, v, dst), a.GoField, dst)
		} else {
			code = setterCode(a, "body", v, patch)
		}
		if code == "" {
			continue
		}
		if patch {
			// Only changed attributes are sent: NetBox rejects re-sent values for
			// some fields (port mappings) and this keeps PATCH bodies minimal.
			code = fmt.Sprintf("if !%s.Equal(state.%s) {\n%s}\n", v, field(a), code)
		}
		b.WriteString(code)
	}
	b.WriteString("return body\n}\n\n")
	return b.String(), nil
}

// ---- API -> Terraform ----

// modelForPrior is the model type name used by readExpr for prior lookups; it
// is set by fromAPIFunc before rendering attributes.
var modelForPrior string

// readExpr returns the expression converting the API field to the model value,
// or "" when the attribute needs statement-level code (nested lists).
func readExpr(a model.Attr, obj string, dataSource bool) string {
	get := obj + ".Get" + a.ReadGoField
	switch a.ReadKind {
	case model.ReadAbsent:
		return ""
	case model.ReadChoice:
		if a.ReadChoiceInt {
			return "conv.ChoiceInt(" + get + "Ok())"
		}
		return "conv.Choice(" + get + "Ok())"
	case model.ReadBrief:
		return "conv.BriefID(" + get + "Ok())"
	case model.ReadBriefList:
		return "conv.BriefIDs(" + get + "())"
	case model.ReadTags:
		return "conv.TagsFromAPI(" + get + "())"
	case model.ReadMap:
		if a.Kind == model.KindCustomFieldsJSON {
			return "customfields.InferJSON(" + get + "())"
		}
		if a.Kind == model.KindCustomFields {
			return "customfields.FromAPI(ctx, " + get + "(), priorCustomFields, diags)"
		}
		if !dataSource && !a.Required {
			return "conv.JSONFromAPIWithPrior(" + get + "(), conv.PriorJSON(prior, func(m *" + modelForPrior + ") jsontypes.Normalized { return m." + field(a) + " }))"
		}
		return "conv.JSONFromAPI(" + get + "())"
	case model.ReadJSON:
		switch a.Kind {
		case model.KindString, model.KindChoice:
			if a.DefaultEmptyString {
				return "conv.AnyOrEmpty(" + get + "Ok())"
			}
			return "conv.Any(" + get + "Ok())"
		}
		if !dataSource && !a.Required {
			return "conv.JSONFromAPIWithPrior(" + get + "(), conv.PriorJSON(prior, func(m *" + modelForPrior + ") jsontypes.Normalized { return m." + field(a) + " }))"
		}
		return "conv.JSONFromAPI(" + get + "())"
	case model.ReadIntList:
		if a.Int64 {
			return "conv.Int64SetFrom64(" + get + "())"
		}
		return "conv.Int64Set(" + get + "())"
	case model.ReadStringList:
		if a.OrderedList {
			return "conv.StringList(" + get + "())"
		}
		return "conv.StringSet(" + get + "())"
	case model.ReadIntRange:
		return "conv.IntRangesFromAPI(" + get + "())"
	case model.ReadAnyList:
		return "conv.AnyListFromAPI(" + get + "())"
	case model.ReadNestedList:
		return ""
	}
	// scalar
	switch a.Kind {
	case model.KindString:
		if a.DefaultEmptyString {
			return "conv.StringOrEmpty(" + get + "Ok())"
		}
		return "conv.String(" + get + "Ok())"
	case model.KindChoice:
		return "conv.ChoiceScalar(" + get + "Ok())"
	case model.KindInt, model.KindChoiceInt, model.KindFK:
		if a.Int64 {
			return "conv.Int64From64(" + get + "Ok())"
		}
		return "conv.Int64From32(" + get + "Ok())"
	case model.KindFloat:
		if a.Float32 {
			return "conv.Float64From32(" + get + "Ok())"
		}
		if !dataSource && !a.ReadOnly && modelForPrior != "" {
			return fmt.Sprintf("conv.Float64Keep(conv.Float64From(%sOk()), conv.PriorFloat(prior, func(m *%s) types.Float64 { return m.%s }), %d)", get, modelForPrior, field(a), a.Precision)
		}
		return "conv.Float64From(" + get + "Ok())"
	case model.KindBool:
		return "conv.Bool(" + get + "Ok())"
	case model.KindDateTime:
		return "conv.RFC3339(" + get + "Ok())"
	}
	return ""
}

// fromAPIFunc renders siteFromAPI (resource) or siteDataFromAPI (data source).
func fromAPIFunc(r *model.Resource, dataSource bool) string {
	var b strings.Builder
	name := lower(r.GoName) + "FromAPI"
	modelName := r.GoName + "Model"
	attrs := r.Attrs
	if dataSource {
		name = lower(r.GoName) + "DataFromAPI"
		modelName = r.GoName + "DataModel"
		attrs = append(append([]model.Attr{}, r.Attrs...), r.ReadOnlyAttrs...)
		for i := range attrs {
			if attrs[i].Kind == model.KindCustomFields {
				attrs[i].Kind = model.KindCustomFieldsJSON
			}
		}
	}
	modelForPrior = modelName
	fmt.Fprintf(&b, "// %s copies an API object into the model.", name)
	if !dataSource {
		b.WriteString(" prior carries the previous state or plan (may be nil).")
	}
	b.WriteString("\n")
	if dataSource {
		fmt.Fprintf(&b, "func %s(ctx context.Context, obj *netbox.%s, out *%s, diags *diag.Diagnostics) {\n", name, r.ReadType, modelName)
	} else {
		fmt.Fprintf(&b, "func %s(ctx context.Context, obj *netbox.%s, prior *%s, out *%s, diags *diag.Diagnostics) {\n", name, r.ReadType, modelName, modelName)
		if r.HasCustomFields {
			b.WriteString("priorCustomFields := types.DynamicNull()\nif prior != nil {\npriorCustomFields = prior.CustomFields\n}\n")
		}
	}
	b.WriteString("_ = ctx\n")
	for _, a := range attrs {
		f := "out." + field(a)
		if a.Name == "id" {
			fmt.Fprintf(&b, "%s = types.Int64Value(int64(obj.GetId()))\n", f)
			continue
		}
		if a.Kind == model.KindNestedList {
			b.WriteString(nestedListRead(r, a, "obj", f, dataSource))
			continue
		}
		expr := readExpr(a, "obj", dataSource)
		if expr == "" {
			if !dataSource {
				fmt.Fprintf(&b, "if prior != nil {\n%s = prior.%s\n}\n", f, field(a))
			}
			continue
		}
		if !dataSource {
			priorExpr := fmt.Sprintf("conv.Prior%s(prior, func(m *%s) %s { return m.%s })", priorKind(a), modelName, tfType(a), field(a))
			switch {
			case a.KeepPriorWhenNull:
				expr = fmt.Sprintf("conv.KeepWhenNull(%s, %s)", expr, priorExpr)
			case a.Kind == model.KindString && !a.ReadOnly:
				expr = fmt.Sprintf("conv.StringKeep(%s, %s, %t)", expr, priorExpr, a.FoldCase)
			}
		}
		fmt.Fprintf(&b, "%s = %s\n", f, expr)
	}
	b.WriteString("}\n\n")
	return b.String()
}

// nestedListRead renders statements reading a nested list into f.
func nestedListRead(r *model.Resource, a model.Attr, obj, f string, dataSource bool) string {
	var b strings.Builder
	it := itemType(r, a)
	if a.ReadKind != model.ReadNestedList || a.NestedRspType == "" {
		if !dataSource {
			fmt.Fprintf(&b, "if prior != nil {\n%s = prior.%s\n}\n", f, field(a))
		}
		return b.String()
	}
	b.WriteString("{\n")
	fmt.Fprintf(&b, "items := %s.Get%s()\n", obj, a.ReadGoField)
	fmt.Fprintf(&b, "vals := make([]%s, 0, len(items))\n", it)
	b.WriteString("for i := range items {\nsrc := &items[i]\n")
	fmt.Fprintf(&b, "vals = append(vals, %s{\n", it)
	for _, n := range a.Nested {
		expr := readExpr(n, "src", dataSource)
		if expr == "" {
			expr = nullExpr(n)
		}
		fmt.Fprintf(&b, "%s: %s,\n", field(n), expr)
	}
	b.WriteString("})\n}\n")
	prior := "nil"
	if !dataSource {
		prior = "priorVal"
		fmt.Fprintf(&b, "var priorVal attr.Value\nif prior != nil {\npriorVal = prior.%s\n}\n", field(a))
	}
	fmt.Fprintf(&b, "%s = conv.ObjectList(ctx, %sAttrTypes, vals, %s, diags)\n", f, lower(it), prior)
	b.WriteString("}\n")
	return b.String()
}

// priorKind names the conv.Prior* helper for an attribute's model type.
func priorKind(a model.Attr) string {
	switch tfType(a) {
	case "types.String":
		return "String"
	case "types.Float64":
		return "Float"
	case "jsontypes.Normalized":
		return "JSON"
	}
	return "Value"
}

func nullExpr(a model.Attr) string {
	switch tfType(a) {
	case "types.String":
		return "types.StringNull()"
	case "types.Int64":
		return "types.Int64Null()"
	case "types.Float64":
		return "types.Float64Null()"
	case "types.Bool":
		return "types.BoolNull()"
	case "timetypes.RFC3339":
		return "timetypes.NewRFC3339Null()"
	case "jsontypes.Normalized":
		return "jsontypes.NewNormalizedNull()"
	case "types.Dynamic":
		return "types.DynamicNull()"
	case "types.Set":
		return "types.SetNull(types.StringType)"
	case "types.List":
		return "types.ListNull(types.StringType)"
	}
	return "types.StringNull()"
}

// sortedFilterNames returns the filter names sorted (for binary search).
func sortedFilterNames(r *model.Resource) []string {
	names := make([]string, 0, len(r.Filters))
	for _, f := range r.Filters {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return names
}
