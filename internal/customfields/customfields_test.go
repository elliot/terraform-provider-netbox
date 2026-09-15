package customfields

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func obj(attrs map[string]attr.Value) types.Dynamic {
	typs := map[string]attr.Type{}
	for k, v := range attrs {
		typs[k] = v.Type(context.Background())
	}
	return types.DynamicValue(types.ObjectValueMust(typs, attrs))
}

func num(f float64) types.Number { return types.NumberValue(big.NewFloat(f)) }

// attrsOf returns the attributes of a dynamic object value.
func attrsOf(t *testing.T, v types.Dynamic) map[string]attr.Value {
	t.Helper()
	o, ok := v.UnderlyingValue().(types.Object)
	if !ok {
		t.Fatalf("expected object, got %T", v.UnderlyingValue())
	}
	return o.Attributes()
}

func str(t *testing.T, v attr.Value) string {
	t.Helper()
	s, ok := v.(types.String)
	if !ok {
		t.Fatalf("expected string, got %T", v)
	}
	return s.ValueString()
}

func intOf(t *testing.T, v attr.Value) int64 {
	t.Helper()
	n, ok := v.(types.Number)
	if !ok {
		t.Fatalf("expected number, got %T", v)
	}
	i, _ := n.ValueBigFloat().Int64()
	return i
}

func TestToAPI(t *testing.T) {
	cfg := obj(map[string]attr.Value{
		"cost_center": types.StringValue("CC-42"),
		"vlan_id":     num(5),
		"ratio":       num(1.25),
		"enabled":     types.BoolValue(true),
		"peers":       types.TupleValueMust([]attr.Type{types.NumberType, types.NumberType}, []attr.Value{num(3), num(7)}),
		"extra":       types.ObjectValueMust(map[string]attr.Type{"a": types.StringType}, map[string]attr.Value{"a": types.StringValue("x")}),
		"cleared":     types.StringNull(),
	})
	var diags diag.Diagnostics
	m := ToAPI(context.Background(), cfg, nil, "", &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if m["cost_center"] != "CC-42" || m["vlan_id"] != int64(5) || m["ratio"] != 1.25 || m["enabled"] != true {
		t.Fatalf("scalars wrong: %#v", m)
	}
	if peers, ok := m["peers"].([]any); !ok || len(peers) != 2 || peers[0] != int64(3) {
		t.Fatalf("peers wrong: %#v", m["peers"])
	}
	if extra, ok := m["extra"].(map[string]any); !ok || extra["a"] != "x" {
		t.Fatalf("extra wrong: %#v", m["extra"])
	}
	if v, present := m["cleared"]; !present || v != nil {
		t.Fatalf("null must be sent as JSON null: %#v", m)
	}
	if ToAPI(context.Background(), types.DynamicNull(), nil, "", &diags) != nil {
		t.Fatal("null config must produce nil map")
	}
}

func TestFromAPIShapesAfterConfig(t *testing.T) {
	api := map[string]any{
		"cost_center": "CC-42",
		"vlan_id":     float64(5),
		"tier":        map[string]any{"value": "gold", "label": "Gold"},
		"site":        map[string]any{"id": float64(30), "url": "https://x/api/dcim/sites/30/", "display": "S", "name": "S", "slug": "s"},
		"sites":       []any{map[string]any{"id": float64(30), "url": "u", "display": "a"}, map[string]any{"id": float64(24), "url": "u", "display": "b"}},
		"extra":       map[string]any{"a": []any{float64(1), float64(2)}, "b": "x"},
		"unmanaged":   "ignored",
		"gone":        nil,
	}
	cfg := obj(map[string]attr.Value{
		"cost_center": types.StringValue("CC-42"),
		"vlan_id":     num(5),
		"tier":        types.StringValue("gold"),
		"site":        num(30),
		"sites":       types.TupleValueMust([]attr.Type{types.NumberType, types.NumberType}, []attr.Value{num(30), num(24)}),
		"extra": types.ObjectValueMust(map[string]attr.Type{
			"a": types.TupleType{ElemTypes: []attr.Type{types.NumberType, types.NumberType}},
			"b": types.StringType,
		}, map[string]attr.Value{
			"a": types.TupleValueMust([]attr.Type{types.NumberType, types.NumberType}, []attr.Value{num(1), num(2)}),
			"b": types.StringValue("x"),
		}),
		"gone": types.StringValue("was-set"),
	})
	var diags diag.Diagnostics
	got := FromAPI(context.Background(), api, cfg, &diags)
	if diags.HasError() {
		t.Fatal(diags)
	}
	attrs := attrsOf(t, got)
	if _, tracked := attrs["unmanaged"]; tracked {
		t.Fatal("unconfigured keys must not be tracked")
	}
	if str(t, attrs["tier"]) != "gold" {
		t.Fatalf("select not unwrapped: %v", attrs["tier"])
	}
	if intOf(t, attrs["site"]) != 30 {
		t.Fatalf("object not reduced to id: %v", attrs["site"])
	}
	if !attrs["gone"].IsNull() {
		t.Fatalf("missing field must be null: %v", attrs["gone"])
	}
	// The shaped value must have exactly the configured type so plans stay stable.
	if !got.UnderlyingValue().Type(context.Background()).Equal(cfg.UnderlyingValue().Type(context.Background())) {
		t.Fatalf("type mismatch:\n got %s\nwant %s", got.UnderlyingValue().Type(context.Background()), cfg.UnderlyingValue().Type(context.Background()))
	}
	want := attrsOf(t, cfg)
	for k, v := range attrs {
		if k != "gone" && !v.Equal(want[k]) {
			t.Fatalf("%s: got %v want %v", k, v, want[k])
		}
	}
	if !FromAPI(context.Background(), api, types.DynamicNull(), &diags).IsNull() {
		t.Fatal("null config must stay null")
	}
}

func TestInfer(t *testing.T) {
	api := map[string]any{
		"n":    float64(3),
		"s":    "x",
		"b":    true,
		"sel":  map[string]any{"value": "a", "label": "A"},
		"obj":  map[string]any{"id": float64(9), "url": "u", "display": "d"},
		"list": []any{float64(1), "two"},
		"nil":  nil,
	}
	got := Infer(api)
	o := attrsOf(t, got)
	if str(t, o["sel"]) != "a" || str(t, o["s"]) != "x" {
		t.Fatalf("strings wrong: %v", o)
	}
	if intOf(t, o["obj"]) != 9 {
		t.Fatalf("object id wrong: %v", o["obj"])
	}
	if !o["nil"].IsNull() {
		t.Fatal("nil must be null")
	}
	if _, ok := o["list"].(types.Tuple); !ok {
		t.Fatalf("list must be a tuple, got %T", o["list"])
	}
	if !Infer(nil).IsNull() {
		t.Fatal("empty map must be null")
	}
}
