package render

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elliot/terraform-provider-netbox/internal/gen/model"
)

func widgetModel(deprecated string, version int64) *model.Resource {
	return &model.Resource{
		App: "dcim", PathSegment: "widgets", Path: "/api/dcim/widgets/", Name: "widget", Plural: "widgets", GoName: "Widget",
		ReadSchema: "Widget", CreateSchema: "WidgetRequest", PatchSchema: "PatchedWidgetRequest",
		ReadType: "Widget", CreateType: "WidgetRequest", PatchType: "PatchedWidgetRequest",
		Service: "DcimAPI", OpPrefix: "DcimWidgets", CreateRequired: []string{"name"},
		Deprecated: deprecated, SchemaVersion: version,
		Attrs: []model.Attr{
			{Name: "id", JSON: "id", GoField: "Id", Kind: model.KindInt, ReadOnly: true, Computed: true, ReadKind: model.ReadScalar, ReadGoField: "Id"},
			{Name: "name", JSON: "name", GoField: "Name", Kind: model.KindString, Required: true, ReadKind: model.ReadScalar, ReadGoField: "Name"},
			{Name: "legacy", JSON: "legacy", GoField: "Legacy", Kind: model.KindString, Nullable: true, ReadKind: model.ReadScalar, ReadGoField: "Legacy",
				Description: "Old field.", Deprecated: "Use name instead."},
			{Name: "items", JSON: "items", GoField: "Items", Kind: model.KindNestedList, ReadKind: model.ReadNestedList, ReadGoField: "Items",
				NestedReqType: "WidgetItemRequest", NestedRspType: "WidgetItem",
				Nested: []model.Attr{
					{Name: "weight", JSON: "weight", GoField: "Weight", Kind: model.KindInt, ReadKind: model.ReadScalar, ReadGoField: "Weight", Deprecated: "Weights are ignored."},
				}},
		},
		ReadOnlyAttrs: []model.Attr{
			{Name: "legacy_count", JSON: "legacy_count", GoField: "LegacyCount", ReadGoField: "LegacyCount", Kind: model.KindInt, ReadKind: model.ReadScalar, ReadOnly: true, Computed: true, Deprecated: "Counts are gone."},
		},
	}
}

func mustParse(t *testing.T, name, src string) {
	t.Helper()
	if _, err := parser.ParseFile(token.NewFileSet(), name, src, 0); err != nil {
		t.Fatalf("%s does not parse: %v\n%s", name, err, src)
	}
}

func TestRenderDeprecation(t *testing.T) {
	r := widgetModel("Use netbox_gadget instead.", 0)
	res, err := resourceFile(r, "4.7.x")
	if err != nil {
		t.Fatal(err)
	}
	mustParse(t, "resource", res)
	ds := dataSourceFile(r, "4.7.x")
	mustParse(t, "data source", ds)

	for _, want := range []string{
		`DeprecationMessage: "Use name instead.",`,
		`DeprecationMessage: "Weights are ignored.",`,
		`DeprecationMessage: "Use netbox_gadget instead.",`,
		`Old field. **Deprecated:** Use name instead.`,
		`\n\n~> **Deprecated:** Use netbox_gadget instead.`,
	} {
		if !strings.Contains(res, want) {
			t.Errorf("resource lacks %s", want)
		}
		if !strings.Contains(ds, want) {
			t.Errorf("data sources lack %s", want)
		}
	}
	// Singular and list data source schemas are both deprecated.
	if n := strings.Count(ds, `DeprecationMessage: "Use netbox_gadget instead.",`); n != 2 {
		t.Errorf("data source file has %d resource-level deprecations, want 2", n)
	}
	if !strings.Contains(ds, `DeprecationMessage: "Counts are gone.",`) {
		t.Error("data-source-only attribute lacks its deprecation")
	}
	if strings.Contains(res, "Counts are gone.") {
		t.Error("data-source-only attribute leaked into the resource")
	}
	for _, unwanted := range []string{"Version:", "UpgradeState", "upgrade."} {
		if strings.Contains(res, unwanted) {
			t.Errorf("version 0 resource contains %q", unwanted)
		}
	}

	plain, err := resourceFile(widgetModel("", 0), "4.7.x")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain, "~> **Deprecated:**") || strings.Count(plain, "DeprecationMessage") != 2 {
		t.Error("a current resource must only carry its attribute deprecations")
	}
}

func TestRenderSchemaVersion(t *testing.T) {
	res, err := resourceFile(widgetModel("", 3), "4.7.x")
	if err != nil {
		t.Fatal(err)
	}
	mustParse(t, "resource", res)
	for _, want := range []string{
		"Version: 3,",
		"var _ resource.ResourceWithUpgradeState = &WidgetResource{}",
		"func (r *WidgetResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {",
		`return upgrade.For("netbox_widget", 3)`,
	} {
		if !strings.Contains(res, want) {
			t.Errorf("resource lacks %s", want)
		}
	}
}

func TestResourcesDocDeprecation(t *testing.T) {
	dir := t.TempDir()
	if err := writeResourcesDoc([]*model.Resource{widgetModel("Use netbox_gadget instead.", 0)}, dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "RESOURCES.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "deprecated: Use netbox_gadget instead.; deprecated attributes: `legacy`, `items.weight`, `legacy_count` |"
	if !strings.Contains(string(b), want) {
		t.Errorf("RESOURCES.md lacks %q:\n%s", want, b)
	}
}
