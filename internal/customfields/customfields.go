// Package customfields converts NetBox custom field values between the
// Terraform dynamic attribute `custom_fields` and the API's free-form map,
// and caches custom field definitions per object type for validation.
//
// Users write native HCL values:
//
//	custom_fields = {
//	  cost_center = "CC-42"      # text / select / date / url
//	  vlan_id     = 5            # integer / decimal
//	  owner_site  = 12           # object  (ID)
//	  peers       = [3, 7]       # multiobject (IDs) / multiselect (values)
//	  extra       = { a = [1] }  # json
//	}
//
// On read, values are coerced to the type the configuration used: selection
// values come back from NetBox as {value,label} objects, related objects as
// {id,url,display,...} objects, and both are reduced to the configured
// scalar. Only keys present in the configuration are tracked, so fields
// managed outside Terraform never produce drift.
package customfields

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/elliot/terraform-provider-netbox/netbox"
)

// Definition is a NetBox custom field definition.
type Definition struct {
	Name              string
	Type              string // text, longtext, integer, decimal, boolean, date, datetime, url, json, select, multiselect, object, multiobject
	RelatedObjectType string
	Required          bool
	Label             string
}

// Cache lazily loads custom field definitions per object type ("dcim.site").
type Cache struct {
	client *netbox.APIClient
	mu     sync.Mutex
	byType map[string]map[string]Definition
}

// NewCache returns a cache backed by the given API client.
func NewCache(client *netbox.APIClient) *Cache {
	return &Cache{client: client, byType: map[string]map[string]Definition{}}
}

// Definitions returns the custom fields defined for objectType, fetching them
// once per provider instance.
func (c *Cache) Definitions(ctx context.Context, objectType string) (map[string]Definition, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if defs, ok := c.byType[objectType]; ok {
		return defs, nil
	}
	defs := map[string]Definition{}
	offset := 0
	for {
		var page netbox.PaginatedCustomFieldList
		url := fmt.Sprintf("/api/extras/custom-fields/?object_type=%s&limit=200&offset=%d", objectType, offset)
		if err := c.client.GetRaw(ctx, url, &page); err != nil {
			return nil, fmt.Errorf("loading custom field definitions for %s: %w", objectType, err)
		}
		for i := range page.Results {
			cf := &page.Results[i]
			d := Definition{Name: cf.GetName(), Required: cf.GetRequired(), Label: cf.GetLabel()}
			if t, ok := cf.GetTypeOk(); ok && t != nil && t.Value.IsSet() && t.Value.Get() != nil {
				d.Type = *t.Value.Get()
			}
			if rot, ok := cf.GetRelatedObjectTypeOk(); ok && rot != nil {
				d.RelatedObjectType = *rot
			}
			defs[d.Name] = d
		}
		offset += len(page.Results)
		if len(page.Results) == 0 || offset >= int(page.GetCount()) {
			break
		}
	}
	c.byType[objectType] = defs
	return defs, nil
}

// Invalidate drops cached definitions (call after custom fields change).
func (c *Cache) Invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byType = map[string]map[string]Definition{}
}

// ---- Terraform -> API ----

// ToAPI converts the configured custom_fields value into the API map. When
// cache and objectType are given, unknown field names are reported as errors
// before anything is sent. Null keys are sent as JSON null (clearing the field).
func ToAPI(ctx context.Context, v types.Dynamic, cache *Cache, objectType string, diags *diag.Diagnostics) map[string]any {
	if v.IsNull() || v.IsUnknown() || v.IsUnderlyingValueNull() || v.IsUnderlyingValueUnknown() {
		return nil
	}
	raw, err := toGo(v.UnderlyingValue())
	if err != nil {
		diags.AddAttributeError(pathCF, "Invalid custom_fields value", err.Error())
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		diags.AddAttributeError(pathCF, "Invalid custom_fields value", fmt.Sprintf("custom_fields must be an object of field name to value, got %T", raw))
		return nil
	}
	if cache != nil && objectType != "" {
		defs, err := cache.Definitions(ctx, objectType)
		if err != nil {
			diags.AddWarning("Could not validate custom field names", err.Error())
		} else if defs != nil {
			var unknown []string
			for k := range m {
				if _, ok := defs[k]; !ok {
					unknown = append(unknown, k)
				}
			}
			if len(unknown) > 0 {
				sort.Strings(unknown)
				known := make([]string, 0, len(defs))
				for k := range defs {
					known = append(known, k)
				}
				sort.Strings(known)
				diags.AddAttributeError(pathCF, "Unknown custom field",
					fmt.Sprintf("No custom field named %s is defined for %s. Defined fields: %s.",
						strings.Join(unknown, ", "), objectType, strings.Join(known, ", ")))
				return nil
			}
		}
	}
	return m
}

// toGo converts a framework value into plain Go data for JSON encoding.
func toGo(v attr.Value) (any, error) {
	if v == nil || v.IsNull() {
		return nil, nil
	}
	if v.IsUnknown() {
		return nil, fmt.Errorf("value is not yet known")
	}
	switch t := v.(type) {
	case types.Dynamic:
		return toGo(t.UnderlyingValue())
	case types.String:
		return t.ValueString(), nil
	case types.Bool:
		return t.ValueBool(), nil
	case types.Number:
		f := t.ValueBigFloat()
		if f.IsInt() {
			if i, acc := f.Int64(); acc == big.Exact {
				return i, nil
			}
		}
		out, _ := f.Float64()
		return out, nil
	case types.Int64:
		return t.ValueInt64(), nil
	case types.Float64:
		return t.ValueFloat64(), nil
	case types.Object:
		out := map[string]any{}
		for k, av := range t.Attributes() {
			g, err := toGo(av)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			out[k] = g
		}
		return out, nil
	case types.Map:
		out := map[string]any{}
		for k, av := range t.Elements() {
			g, err := toGo(av)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			out[k] = g
		}
		return out, nil
	case types.Tuple:
		return listToGo(t.Elements())
	case types.List:
		return listToGo(t.Elements())
	case types.Set:
		return listToGo(t.Elements())
	}
	return nil, fmt.Errorf("unsupported value type %T", v)
}

func listToGo(elems []attr.Value) (any, error) {
	out := make([]any, 0, len(elems))
	for i, e := range elems {
		g, err := toGo(e)
		if err != nil {
			return nil, fmt.Errorf("[%d]: %w", i, err)
		}
		out = append(out, g)
	}
	return out, nil
}

// ---- API -> Terraform ----

// FromAPI shapes the API custom_fields map after the configured value: only
// configured keys are kept and each value is coerced to the configured type.
// A null configuration stays null (nothing is tracked).
func FromAPI(ctx context.Context, m map[string]any, prior types.Dynamic, diags *diag.Diagnostics) types.Dynamic {
	if prior.IsNull() || prior.IsUnderlyingValueNull() {
		return types.DynamicNull()
	}
	if prior.IsUnknown() || prior.IsUnderlyingValueUnknown() {
		// Nothing to shape after: expose every field with inferred types.
		return Infer(m)
	}
	shaped := coerce(m, prior.UnderlyingValue())
	return types.DynamicValue(shaped)
}

// Infer converts an API custom_fields map into a dynamic object with inferred
// types (data sources): objects with an id become their id, {value,label}
// objects become the value, lists are tuples.
func Infer(m map[string]any) types.Dynamic {
	if len(m) == 0 {
		return types.DynamicNull()
	}
	return types.DynamicValue(inferValue(m))
}

// InferJSON returns every custom field as a normalized JSON string with
// selection and related-object values unwrapped (data sources, where dynamic
// values cannot be nested inside list items).
func InferJSON(m map[string]any) jsontypes.Normalized {
	if len(m) == 0 {
		return jsontypes.NewNormalizedNull()
	}
	out := map[string]any{}
	for k, v := range m {
		out[k] = unwrap(v)
	}
	data, err := json.Marshal(out)
	if err != nil {
		return jsontypes.NewNormalizedNull()
	}
	return jsontypes.NewNormalizedValue(string(data))
}

// unwrap reduces NetBox's read representations to plain values.
func unwrap(v any) any {
	switch t := v.(type) {
	case map[string]any:
		if id, ok := objectID(t); ok {
			return id
		}
		if val, ok := choiceValue(t); ok {
			return unwrap(val)
		}
		out := map[string]any{}
		for k, e := range t {
			out[k] = unwrap(e)
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, e := range t {
			out = append(out, unwrap(e))
		}
		return out
	}
	return v
}

// inferValue infers a framework value from arbitrary JSON data.
func inferValue(v any) attr.Value {
	switch t := v.(type) {
	case nil:
		return types.StringNull()
	case string:
		return types.StringValue(t)
	case bool:
		return types.BoolValue(t)
	case float64:
		return types.NumberValue(big.NewFloat(t))
	case int64:
		return types.NumberValue(big.NewFloat(float64(t)))
	case int:
		return types.NumberValue(big.NewFloat(float64(t)))
	case []any:
		elems := make([]attr.Value, 0, len(t))
		typs := make([]attr.Type, 0, len(t))
		for _, e := range t {
			ev := inferValue(e)
			elems = append(elems, ev)
			typs = append(typs, ev.Type(context.Background()))
		}
		return types.TupleValueMust(typs, elems)
	case map[string]any:
		if id, ok := objectID(t); ok {
			return types.NumberValue(big.NewFloat(float64(id)))
		}
		if val, ok := choiceValue(t); ok {
			return inferValue(val)
		}
		attrs := map[string]attr.Value{}
		typs := map[string]attr.Type{}
		for k, e := range t {
			ev := inferValue(e)
			attrs[k] = ev
			typs[k] = ev.Type(context.Background())
		}
		return types.ObjectValueMust(typs, attrs)
	}
	return types.StringValue(fmt.Sprint(v))
}

// coerce converts API data into a value of the same type as target.
func coerce(data any, target attr.Value) attr.Value {
	ctx := context.Background()
	switch tv := target.(type) {
	case types.Dynamic:
		if tv.IsUnderlyingValueNull() || tv.IsUnderlyingValueUnknown() {
			return inferValue(data)
		}
		return coerce(data, tv.UnderlyingValue())
	case types.Object:
		m, _ := data.(map[string]any)
		typs := tv.AttributeTypes(ctx)
		attrs := map[string]attr.Value{}
		for k, at := range typs {
			raw, present := m[k]
			if !present || raw == nil {
				attrs[k] = nullOf(at)
				continue
			}
			attrs[k] = coerce(raw, valueOfType(at, tv.Attributes()[k]))
		}
		if obj, d := types.ObjectValue(typs, attrs); !d.HasError() {
			return obj
		}
		return inferValue(data)
	case types.Map:
		m, _ := data.(map[string]any)
		elemT := tv.ElementType(ctx)
		elems := map[string]attr.Value{}
		for k, raw := range m {
			if raw == nil {
				elems[k] = nullOf(elemT)
				continue
			}
			elems[k] = coerce(raw, nullOf(elemT))
		}
		if mv, d := types.MapValue(elemT, elems); !d.HasError() {
			return mv
		}
		return inferValue(data)
	case types.Tuple:
		list, _ := data.([]any)
		etypes := tv.ElementTypes(ctx)
		elems := make([]attr.Value, 0, len(list))
		typs := make([]attr.Type, 0, len(list))
		for i, raw := range list {
			var tgt attr.Value
			if i < len(etypes) {
				tgt = valueOfType(etypes[i], tv.Elements()[i])
			} else if len(etypes) > 0 {
				tgt = nullOf(etypes[len(etypes)-1])
			}
			var ev attr.Value
			if tgt == nil {
				ev = inferValue(raw)
			} else {
				ev = coerce(raw, tgt)
			}
			elems = append(elems, ev)
			typs = append(typs, ev.Type(ctx))
		}
		return types.TupleValueMust(typs, elems)
	case types.List:
		list, _ := data.([]any)
		elemT := tv.ElementType(ctx)
		elems := make([]attr.Value, 0, len(list))
		for _, raw := range list {
			elems = append(elems, coerce(raw, nullOf(elemT)))
		}
		if lv, d := types.ListValue(elemT, elems); !d.HasError() {
			return lv
		}
		return inferValue(data)
	case types.Set:
		list, _ := data.([]any)
		elemT := tv.ElementType(ctx)
		elems := make([]attr.Value, 0, len(list))
		for _, raw := range list {
			elems = append(elems, coerce(raw, nullOf(elemT)))
		}
		if sv, d := types.SetValue(elemT, elems); !d.HasError() {
			return sv
		}
		return inferValue(data)
	case types.String:
		return coerceString(data)
	case types.Number:
		return coerceNumber(data)
	case types.Bool:
		switch b := data.(type) {
		case bool:
			return types.BoolValue(b)
		case nil:
			return types.BoolNull()
		}
		return inferValue(data)
	}
	return inferValue(data)
}

func coerceString(data any) attr.Value {
	switch d := data.(type) {
	case nil:
		return types.StringNull()
	case string:
		return types.StringValue(d)
	case map[string]any:
		if val, ok := choiceValue(d); ok {
			return coerceString(val)
		}
		if id, ok := objectID(d); ok {
			return types.StringValue(fmt.Sprintf("%d", id))
		}
	case float64:
		if d == float64(int64(d)) {
			return types.StringValue(fmt.Sprintf("%d", int64(d)))
		}
		return types.StringValue(fmt.Sprint(d))
	case bool:
		return types.StringValue(fmt.Sprint(d))
	}
	return inferValue(data)
}

func coerceNumber(data any) attr.Value {
	switch d := data.(type) {
	case nil:
		return types.NumberNull()
	case float64:
		return types.NumberValue(big.NewFloat(d))
	case int64:
		return types.NumberValue(big.NewFloat(float64(d)))
	case map[string]any:
		if id, ok := objectID(d); ok {
			return types.NumberValue(big.NewFloat(float64(id)))
		}
		if val, ok := choiceValue(d); ok {
			return coerceNumber(val)
		}
	case string:
		f, ok := new(big.Float).SetString(d)
		if ok {
			return types.NumberValue(f)
		}
	}
	return inferValue(data)
}

// objectID reports whether m looks like a nested NetBox object and returns its id.
func objectID(m map[string]any) (int64, bool) {
	id, ok := m["id"].(float64)
	if !ok {
		return 0, false
	}
	if _, hasURL := m["url"]; !hasURL {
		if _, hasDisplay := m["display"]; !hasDisplay {
			return 0, false
		}
	}
	return int64(id), true
}

// choiceValue reports whether m is a {value,label} selection object.
func choiceValue(m map[string]any) (any, bool) {
	if len(m) != 2 {
		return nil, false
	}
	val, hasValue := m["value"]
	_, hasLabel := m["label"]
	if hasValue && hasLabel {
		return val, true
	}
	return nil, false
}

// valueOfType returns v when non-nil, else a null of type t.
func valueOfType(t attr.Type, v attr.Value) attr.Value {
	if v != nil {
		return v
	}
	return nullOf(t)
}

// nullOf returns a null value of the given type.
func nullOf(t attr.Type) attr.Value {
	switch tt := t.(type) {
	case basetypes.StringType:
		return types.StringNull()
	case basetypes.NumberType:
		return types.NumberNull()
	case basetypes.BoolType:
		return types.BoolNull()
	case basetypes.Int64Type:
		return types.Int64Null()
	case basetypes.Float64Type:
		return types.Float64Null()
	case basetypes.DynamicType:
		return types.DynamicNull()
	case basetypes.ListType:
		return types.ListNull(tt.ElemType)
	case basetypes.SetType:
		return types.SetNull(tt.ElemType)
	case basetypes.MapType:
		return types.MapNull(tt.ElemType)
	case basetypes.ObjectType:
		return types.ObjectNull(tt.AttrTypes)
	case basetypes.TupleType:
		return types.TupleNull(tt.ElemTypes)
	}
	return types.DynamicNull()
}
