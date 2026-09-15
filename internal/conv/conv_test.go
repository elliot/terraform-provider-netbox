package conv

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFloat64Keep(t *testing.T) {
	api := 12.345679
	got := Float64Keep(&api, true, types.Float64Value(12.3456789), 6)
	if got.ValueFloat64() != 12.3456789 {
		t.Fatalf("expected prior value kept, got %v", got)
	}
	got = Float64Keep(&api, true, types.Float64Value(12.0), 6)
	if got.ValueFloat64() != 12.345679 {
		t.Fatalf("expected API value, got %v", got)
	}
	if !Float64Keep(nil, false, types.Float64Value(1), 2).IsNull() {
		t.Fatal("expected null")
	}
}

func TestCustomFieldsFromAPI(t *testing.T) {
	m := map[string]any{
		"cost":   "CC-42",
		"sel":    map[string]any{"value": "a", "label": "A"},
		"multi":  []any{map[string]any{"value": "x", "label": "X"}},
		"unused": nil,
	}
	prior := jsontypes.NewNormalizedValue(`{"cost":"","sel":"","multi":[]}`)
	got := CustomFieldsFromAPI(m, prior)
	want := jsontypes.NewNormalizedValue(`{"cost":"CC-42","multi":["x"],"sel":"a"}`)
	eq, diags := got.StringSemanticEquals(t.Context(), want)
	if diags.HasError() || !eq {
		t.Fatalf("got %s want %s", got.ValueString(), want.ValueString())
	}
	if !CustomFieldsFromAPI(m, jsontypes.NewNormalizedNull()).IsNull() {
		t.Fatal("unconfigured custom_fields must stay null")
	}
	if AllCustomFieldsFromAPI(m).IsNull() {
		t.Fatal("data sources expose all custom fields")
	}
}

func TestJSONFromAPIWithPrior(t *testing.T) {
	if !JSONFromAPIWithPrior(map[string]any{}, jsontypes.NewNormalizedNull()).IsNull() {
		t.Fatal("empty object with null prior must stay null")
	}
	if JSONFromAPIWithPrior(map[string]any{"a": 1}, jsontypes.NewNormalizedNull()).IsNull() {
		t.Fatal("non-empty object must be exposed")
	}
	if JSONFromAPIWithPrior(map[string]any{}, jsontypes.NewNormalizedValue(`{}`)).IsNull() {
		t.Fatal("configured empty object must round-trip")
	}
}

func TestChoiceMapping(t *testing.T) {
	empty := ""
	if !ChoiceScalar(&empty, true).IsNull() {
		t.Fatal(`"" choice must map to null`)
	}
	v := "active"
	if ChoiceScalar(&v, true).ValueString() != "active" {
		t.Fatal("choice value lost")
	}
}
