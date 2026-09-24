// Package conv converts between Terraform framework values and the generated
// NetBox client models. It is hand-written and used by generated code.
package conv

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/elliot/terraform-provider-netbox/netbox"
)

// ---- Terraform -> API ----

// Known reports whether a value is neither null nor unknown.
func Known(v attr.Value) bool { return !v.IsNull() && !v.IsUnknown() }

// Int32 converts a known Int64 value to int32 (values out of range are clamped
// and reported by the provider's schema validators before reaching here).
func Int32(v types.Int64) int32 {
	n := v.ValueInt64()
	if n > math.MaxInt32 || n < math.MinInt32 {
		return 0
	}
	return int32(n)
}

// Int32Ptr returns a pointer to the int32 value or nil when null/unknown.
func Int32Ptr(v types.Int64) *int32 {
	if !Known(v) {
		return nil
	}
	n := Int32(v)
	return &n
}

// NullableInt32 wraps an Int64 value into the client's nullable type
// (used for the rare required-but-nullable properties).
func NullableInt32(v types.Int64) netbox.NullableInt32 {
	return *netbox.NewNullableInt32(Int32Ptr(v))
}

// Float32 converts a Float64 value to float32.
func Float32(v types.Float64) float32 { return float32(v.ValueFloat64()) }

// Int32s converts a Set/List of Int64 into []int32 (never nil).
func Int32s(ctx context.Context, v attr.Value, diags *diag.Diagnostics) []int32 {
	out := []int32{}
	for _, n := range int64Elements(ctx, v, diags) {
		out = append(out, Int32(types.Int64Value(n)))
	}
	return out
}

// Int64s converts a Set/List of Int64 into []int64 (never nil).
func Int64s(ctx context.Context, v attr.Value, diags *diag.Diagnostics) []int64 {
	out := []int64{}
	out = append(out, int64Elements(ctx, v, diags)...)
	return out
}

func int64Elements(ctx context.Context, v attr.Value, diags *diag.Diagnostics) []int64 {
	var out []int64
	switch c := v.(type) {
	case types.Set:
		if Known(c) {
			diags.Append(c.ElementsAs(ctx, &out, false)...)
		}
	case types.List:
		if Known(c) {
			diags.Append(c.ElementsAs(ctx, &out, false)...)
		}
	}
	return out
}

// Strings converts a Set/List of String into []string (never nil).
func Strings(ctx context.Context, v attr.Value, diags *diag.Diagnostics) []string {
	out := []string{}
	switch c := v.(type) {
	case types.Set:
		if Known(c) {
			diags.Append(c.ElementsAs(ctx, &out, false)...)
		}
	case types.List:
		if Known(c) {
			diags.Append(c.ElementsAs(ctx, &out, false)...)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}

// TagsToAPI converts a set of tag slugs into nested tag requests.
func TagsToAPI(ctx context.Context, v types.Set, diags *diag.Diagnostics) []netbox.NestedTagRequest {
	slugs := Strings(ctx, v, diags)
	out := make([]netbox.NestedTagRequest, 0, len(slugs))
	for _, s := range slugs {
		out = append(out, *netbox.NewNestedTagRequest(s))
	}
	return out
}

// JSONToAPI decodes a normalized JSON string into an arbitrary value.
func JSONToAPI(v jsontypes.Normalized, diags *diag.Diagnostics) any {
	if !Known(v) {
		return nil
	}
	var out any
	if err := json.Unmarshal([]byte(v.ValueString()), &out); err != nil {
		diags.AddError("Invalid JSON", err.Error())
		return nil
	}
	return out
}

// JSONObjectToAPI decodes a normalized JSON string into a map (custom_fields).
func JSONObjectToAPI(v jsontypes.Normalized, diags *diag.Diagnostics) map[string]any {
	if !Known(v) {
		return nil
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(v.ValueString()), &out); err != nil {
		diags.AddError("Invalid JSON object", err.Error())
		return nil
	}
	return out
}

// JSONStringMapToAPI decodes a normalized JSON object of strings.
func JSONStringMapToAPI(v jsontypes.Normalized, diags *diag.Diagnostics) map[string]string {
	if !Known(v) {
		return nil
	}
	out := map[string]string{}
	if err := json.Unmarshal([]byte(v.ValueString()), &out); err != nil {
		diags.AddError("Invalid JSON object of strings", err.Error())
		return nil
	}
	return out
}

// IntRangesToAPI converts a List(List(Int64)) into [][]int32.
func IntRangesToAPI(ctx context.Context, v types.List, diags *diag.Diagnostics) [][]int32 {
	out := [][]int32{}
	if !Known(v) {
		return out
	}
	var lists []types.List
	diags.Append(v.ElementsAs(ctx, &lists, false)...)
	for _, l := range lists {
		out = append(out, Int32s(ctx, l, diags))
	}
	return out
}

// AnyListToAPI converts a List(List(String)) into [][]any (custom field choice sets).
func AnyListToAPI(ctx context.Context, v types.List, diags *diag.Diagnostics) [][]any {
	out := [][]any{}
	if !Known(v) {
		return out
	}
	var lists []types.List
	diags.Append(v.ElementsAs(ctx, &lists, false)...)
	for _, l := range lists {
		row := []any{}
		for _, s := range Strings(ctx, l, diags) {
			row = append(row, s)
		}
		out = append(out, row)
	}
	return out
}

// Time converts an RFC3339 value to time.Time (zero when null/unknown).
func Time(v timetypes.RFC3339, diags *diag.Diagnostics) time.Time {
	if !Known(v) {
		return time.Time{}
	}
	t, d := v.ValueRFC3339Time()
	diags.Append(d...)
	return t
}

// ---- API -> Terraform ----

// String converts a (*string, ok) getter result; null when absent.
func String(v *string, ok bool) types.String {
	if !ok || v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// StringOrEmpty converts a getter result, mapping absent/null to "".
func StringOrEmpty(v *string, ok bool) types.String {
	if !ok || v == nil {
		return types.StringValue("")
	}
	return types.StringValue(*v)
}

// ChoiceScalar converts a bare-string choice, mapping "" and null to null.
func ChoiceScalar(v *string, ok bool) types.String {
	if !ok || v == nil || *v == "" {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// Choice converts a {value,label} read object to its value ("" -> null).
func Choice(v *netbox.ChoiceString, ok bool) types.String {
	if !ok || v == nil {
		return types.StringNull()
	}
	return ChoiceScalar(v.Value.Get(), v.Value.IsSet())
}

// ChoiceInt converts an integer {value,label} read object to its value.
func ChoiceInt(v *netbox.ChoiceInteger, ok bool) types.Int64 {
	if !ok || v == nil {
		return types.Int64Null()
	}
	return Int64From32(v.Value.Get(), v.Value.IsSet())
}

// Any converts an untyped read value to a string attribute (colors and other
// fields NetBox declares as oneOf strings are untyped on the read side).
func Any(v *any, ok bool) types.String {
	if !ok || v == nil || *v == nil {
		return types.StringNull()
	}
	switch t := (*v).(type) {
	case string:
		return types.StringValue(t)
	default:
		return types.StringValue(fmt.Sprint(t))
	}
}

// AnyOrEmpty is Any mapping null to "".
func AnyOrEmpty(v *any, ok bool) types.String {
	s := Any(v, ok)
	if s.IsNull() {
		return types.StringValue("")
	}
	return s
}

// Int64From32 converts a (*int32, ok) getter result.
func Int64From32(v *int32, ok bool) types.Int64 {
	if !ok || v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

// Int64From64 converts a (*int64, ok) getter result.
func Int64From64(v *int64, ok bool) types.Int64 {
	if !ok || v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

// Float64From converts a (*float64, ok) getter result.
func Float64From(v *float64, ok bool) types.Float64 {
	if !ok || v == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*v)
}

// Float64Keep converts a float getter result while preserving the prior
// (planned) value when it equals the API value at the given number of
// decimals, so NetBox's decimal rounding does not produce perpetual diffs.
// decimals <= 0 means 6.
func Float64Keep(got types.Float64, prior types.Float64, decimals int) types.Float64 {
	if got.IsNull() || !Known(prior) {
		return got
	}
	if decimals <= 0 {
		decimals = 6
	}
	scale := math.Pow(10, float64(decimals))
	if math.Round(prior.ValueFloat64()*scale) == math.Round(got.ValueFloat64()*scale) {
		return prior
	}
	return got
}

// Float64From32 converts a (*float32, ok) getter result.
func Float64From32(v *float32, ok bool) types.Float64 {
	if !ok || v == nil {
		return types.Float64Null()
	}
	return types.Float64Value(float64(*v))
}

// Bool converts a (*bool, ok) getter result.
func Bool(v *bool, ok bool) types.Bool {
	if !ok || v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

// RFC3339 converts a (*time.Time, ok) getter result.
func RFC3339(v *time.Time, ok bool) timetypes.RFC3339 {
	if !ok || v == nil {
		return timetypes.NewRFC3339Null()
	}
	return timetypes.NewRFC3339TimeValue(v.UTC())
}

// idGetter is implemented by every Brief*/Nested* client type.
type idGetter interface{ GetId() int32 }

// BriefID converts a (*BriefX, ok) getter result to the object's ID.
func BriefID[T any, P interface {
	*T
	idGetter
}](v P, ok bool) types.Int64 {
	if !ok || v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(v.GetId()))
}

// BriefIDs converts a slice of Brief*/Nested* objects into a Set of IDs.
func BriefIDs[T any, P interface {
	*T
	idGetter
}](items []T) types.Set {
	vals := make([]attr.Value, 0, len(items))
	for i := range items {
		vals = append(vals, types.Int64Value(int64(P(&items[i]).GetId())))
	}
	return types.SetValueMust(types.Int64Type, vals)
}

// Int64Set converts []int32 into a Set of Int64.
func Int64Set(items []int32) types.Set {
	vals := make([]attr.Value, 0, len(items))
	for _, n := range items {
		vals = append(vals, types.Int64Value(int64(n)))
	}
	return types.SetValueMust(types.Int64Type, vals)
}

// Int64SetFrom64 converts []int64 into a Set of Int64.
func Int64SetFrom64(items []int64) types.Set {
	vals := make([]attr.Value, 0, len(items))
	for _, n := range items {
		vals = append(vals, types.Int64Value(n))
	}
	return types.SetValueMust(types.Int64Type, vals)
}

// StringSet converts []string into a Set of String.
func StringSet(items []string) types.Set {
	vals := make([]attr.Value, 0, len(items))
	for _, s := range items {
		vals = append(vals, types.StringValue(s))
	}
	return types.SetValueMust(types.StringType, vals)
}

// StringList converts []string into a List of String.
func StringList(items []string) types.List {
	vals := make([]attr.Value, 0, len(items))
	for _, s := range items {
		vals = append(vals, types.StringValue(s))
	}
	return types.ListValueMust(types.StringType, vals)
}

// TagsFromAPI converts nested tags into a Set of slugs.
func TagsFromAPI(tags []netbox.NestedTag) types.Set {
	slugs := make([]string, 0, len(tags))
	for i := range tags {
		if s, ok := tags[i].GetSlugOk(); ok && s != nil {
			slugs = append(slugs, *s)
		}
	}
	return StringSet(slugs)
}

// JSONFromAPI encodes an arbitrary API value as normalized JSON (null when nil).
func JSONFromAPI(v any) jsontypes.Normalized {
	if v == nil {
		return jsontypes.NewNormalizedNull()
	}
	data, err := json.Marshal(v)
	if err != nil {
		return jsontypes.NewNormalizedNull()
	}
	return jsontypes.NewNormalizedValue(string(data))
}

// JSONFromAPIWithPrior is JSONFromAPI for optional JSON attributes: when the
// attribute was not configured (prior null) and NetBox returns an empty
// object/array or null, the attribute stays null so plans remain stable.
func JSONFromAPIWithPrior(v any, prior jsontypes.Normalized) jsontypes.Normalized {
	if prior.IsNull() || prior.IsUnknown() {
		switch t := v.(type) {
		case nil:
			return jsontypes.NewNormalizedNull()
		case map[string]any:
			if len(t) == 0 {
				return jsontypes.NewNormalizedNull()
			}
		case []any:
			if len(t) == 0 {
				return jsontypes.NewNormalizedNull()
			}
		case map[string]string:
			if len(t) == 0 {
				return jsontypes.NewNormalizedNull()
			}
		}
	}
	return JSONFromAPI(v)
}

// IntRangesFromAPI converts [][]int32 into a List(List(Int64)).
func IntRangesFromAPI(ranges [][]int32) types.List {
	elem := types.ListType{ElemType: types.Int64Type}
	vals := make([]attr.Value, 0, len(ranges))
	for _, r := range ranges {
		inner := make([]attr.Value, 0, len(r))
		for _, n := range r {
			inner = append(inner, types.Int64Value(int64(n)))
		}
		vals = append(vals, types.ListValueMust(types.Int64Type, inner))
	}
	return types.ListValueMust(elem, vals)
}

// AnyListFromAPI converts [][]any into a List(List(String)).
func AnyListFromAPI(rows [][]any) types.List {
	elem := types.ListType{ElemType: types.StringType}
	vals := make([]attr.Value, 0, len(rows))
	for _, r := range rows {
		inner := make([]attr.Value, 0, len(r))
		for _, c := range r {
			inner = append(inner, types.StringValue(fmt.Sprint(c)))
		}
		vals = append(vals, types.ListValueMust(types.StringType, inner))
	}
	return types.ListValueMust(elem, vals)
}

// ObjectList builds a List of objects from a slice of tfsdk-tagged models.
// An empty slice becomes null when prior is null so that omitted optional
// nested lists do not drift.
func ObjectList[T any](ctx context.Context, attrTypes map[string]attr.Type, items []T, prior attr.Value, diags *diag.Diagnostics) types.List {
	objType := types.ObjectType{AttrTypes: attrTypes}
	if len(items) == 0 && (prior == nil || prior.IsNull()) {
		return types.ListNull(objType)
	}
	l, d := types.ListValueFrom(ctx, objType, items)
	diags.Append(d...)
	return l
}

// ---- Data source helpers ----

// Filter is one generic query filter for list endpoints.
type Filter struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// FilterAttrTypes is the object type of a Filter.
var FilterAttrTypes = map[string]attr.Type{"name": types.StringType, "value": types.StringType}

// ApplyFilters adds the configured filters to q, validating names against allowed.
func ApplyFilters(ctx context.Context, filters types.List, allowed []string, q url.Values, diags *diag.Diagnostics) {
	if !Known(filters) {
		return
	}
	var fs []Filter
	diags.Append(filters.ElementsAs(ctx, &fs, false)...)
	for _, f := range fs {
		// Names are validated at plan time by FilterName; this catches values
		// that were unknown then.
		name := f.Name.ValueString()
		if !validFilter(allowed, name) {
			diags.AddError("Unknown filter", unknownFilterDetail(name, allowed))
			continue
		}
		q.Add(name, f.Value.ValueString())
	}
}

// StringKeep returns got unless prior (the planned/configured value) is
// equivalent after NetBox's normalisation: surrounding whitespace is trimmed by
// DRF and, when fold is set, letter case is ignored (MAC addresses, WWNs).
func StringKeep(got types.String, prior types.String, fold bool) types.String {
	if got.IsNull() || !Known(prior) {
		return got
	}
	a, b := strings.TrimSpace(prior.ValueString()), got.ValueString()
	if a == b || (fold && strings.EqualFold(a, b)) {
		return prior
	}
	return got
}

// PriorString returns the string attribute selected by get from prior, or
// null when prior is nil (used by generated FromAPI functions).
func PriorString[M any](prior *M, get func(*M) types.String) types.String {
	if prior == nil {
		return types.StringNull()
	}
	return get(prior)
}

// KeepWhenNull returns prior when got is null and prior is known: used for
// values NetBox returns only once (token secrets) or never echoes.
func KeepWhenNull[V attr.Value](got V, prior V) V {
	if got.IsNull() && !prior.IsNull() && !prior.IsUnknown() {
		return prior
	}
	return got
}

// PriorFloat returns the float attribute selected by get from prior, or null
// when prior is nil (used by generated FromAPI functions).
func PriorFloat[M any](prior *M, get func(*M) types.Float64) types.Float64 {
	if prior == nil {
		return types.Float64Null()
	}
	return get(prior)
}

// PriorJSON returns the JSON attribute selected by get from prior, or null
// when prior is nil (used by generated FromAPI functions).
func PriorJSON[M any](prior *M, get func(*M) jsontypes.Normalized) jsontypes.Normalized {
	if prior == nil {
		return jsontypes.NewNormalizedNull()
	}
	return get(prior)
}

// PriorValue returns the attribute selected by get from prior; when prior is
// nil the zero value of V (a null value for framework types) is returned via zero.
func PriorValue[M any, V attr.Value](prior *M, get func(*M) V) V {
	var zero V
	if prior == nil {
		return zero
	}
	return get(prior)
}
