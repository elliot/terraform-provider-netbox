package build

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/elliot/terraform-provider-netbox/internal/gen/model"
	"github.com/elliot/terraform-provider-netbox/internal/gen/openapi"
	"github.com/elliot/terraform-provider-netbox/internal/gen/overrides"
)

// widgetSpec is a minimal NetBox-shaped document with one CRUD collection
// (/api/dcim/widgets/). "legacy" and "old_ports" are deprecated in the spec,
// "items" is a nested list whose "weight" is deprecated, and "legacy_count"
// is a deprecated read-only property.
const widgetSpec = `{
  "info": {"version": "4.7.4"},
  "paths": {
    "/api/dcim/widgets/": {
      "get":  {"operationId": "dcim_widgets_list", "tags": ["dcim"], "description": "Get a list of widget objects.",
               "parameters": [{"name": "name", "in": "query"}],
               "responses": {"200": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/PaginatedWidgetList"}}}}}},
      "post": {"operationId": "dcim_widgets_create", "tags": ["dcim"],
               "requestBody": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/WidgetRequest"}}}},
               "responses": {"201": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Widget"}}}}}}
    },
    "/api/dcim/widgets/{id}/": {
      "get":    {"operationId": "dcim_widgets_retrieve", "tags": ["dcim"],
                 "responses": {"200": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Widget"}}}}}},
      "patch":  {"operationId": "dcim_widgets_partial_update", "tags": ["dcim"],
                 "requestBody": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/WidgetRequest"}}}},
                 "responses": {"200": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Widget"}}}}}},
      "delete": {"operationId": "dcim_widgets_destroy", "tags": ["dcim"], "responses": {"204": {}}}
    }
  },
  "components": {"schemas": {
    "WidgetRequest": {"type": "object", "required": ["name"], "properties": {
      "name": {"type": "string", "maxLength": 100},
      "legacy": {"type": "string", "nullable": true, "deprecated": true},
      "old_ports": {"type": "array", "items": {"type": "integer"}, "nullable": true, "deprecated": true},
      "description": {"type": "string"},
      "items": {"type": "array", "items": {"$ref": "#/components/schemas/WidgetItemRequest"}}
    }},
    "WidgetItemRequest": {"type": "object", "required": ["label"], "properties": {
      "label": {"type": "string"},
      "weight": {"type": "integer", "deprecated": true}
    }},
    "Widget": {"type": "object", "properties": {
      "id": {"type": "integer", "readOnly": true},
      "url": {"type": "string", "readOnly": true},
      "name": {"type": "string"},
      "legacy": {"type": "string", "nullable": true, "deprecated": true},
      "old_ports": {"type": "array", "items": {"type": "integer"}, "nullable": true, "deprecated": true},
      "description": {"type": "string"},
      "items": {"type": "array", "items": {"$ref": "#/components/schemas/WidgetItem"}},
      "legacy_count": {"type": "integer", "readOnly": true, "deprecated": true},
      "widget_count": {"type": "integer", "readOnly": true}
    }},
    "WidgetItem": {"type": "object", "properties": {
      "label": {"type": "string"},
      "weight": {"type": "integer", "deprecated": true}
    }},
    "PaginatedWidgetList": {"type": "object", "properties": {"results": {"type": "array", "items": {"$ref": "#/components/schemas/Widget"}}}}
  }}
}`

func buildWidget(t *testing.T, overridesYAML string, specEdit func(*openapi.Document)) (*model.Resource, error) {
	t.Helper()
	var doc openapi.Document
	if err := json.Unmarshal([]byte(widgetSpec), &doc); err != nil {
		t.Fatal(err)
	}
	if specEdit != nil {
		specEdit(&doc)
	}
	var f overrides.File
	if err := yaml.Unmarshal([]byte(overridesYAML), &f); err != nil {
		t.Fatal(err)
	}
	if f.Resources == nil {
		f.Resources = map[string]*overrides.Resource{}
	}
	rs, _, err := Build(&doc, f.Resources)
	if err != nil {
		return nil, err
	}
	if len(rs) != 1 {
		t.Fatalf("got %d resources", len(rs))
	}
	return rs[0], nil
}

func attrIn(attrs []model.Attr, name string) *model.Attr {
	for i := range attrs {
		if attrs[i].Name == name {
			return &attrs[i]
		}
	}
	return nil
}

func TestDeprecationFromSpec(t *testing.T) {
	r, err := buildWidget(t, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"legacy", "old_ports"} {
		if a := r.Attr(n); a == nil || a.Deprecated != DefaultAttributeDeprecation {
			t.Errorf("%s: want the default deprecation, got %+v", n, a)
		}
	}
	for _, n := range []string{"name", "description", "items", "id", "url"} {
		if a := r.Attr(n); a == nil || a.Deprecated != "" {
			t.Errorf("%s: must not be deprecated, got %+v", n, a)
		}
	}
	items := r.Attr("items")
	if w := attrIn(items.Nested, "weight"); w == nil || w.Deprecated != DefaultAttributeDeprecation {
		t.Errorf("nested weight: want the default deprecation, got %+v", w)
	}
	if l := attrIn(items.Nested, "label"); l == nil || l.Deprecated != "" {
		t.Errorf("nested label: must not be deprecated, got %+v", l)
	}
	if a := attrIn(r.ReadOnlyAttrs, "legacy_count"); a == nil || a.Deprecated != DefaultAttributeDeprecation {
		t.Errorf("read-only legacy_count: want the default deprecation, got %+v", a)
	}
	if a := attrIn(r.ReadOnlyAttrs, "widget_count"); a == nil || a.Deprecated != "" {
		t.Errorf("widget_count: must not be deprecated, got %+v", a)
	}
	if r.Deprecated != "" || r.SchemaVersion != 0 {
		t.Errorf("resource: deprecated=%q version=%d, want neither", r.Deprecated, r.SchemaVersion)
	}
}

func TestDeprecationOverrides(t *testing.T) {
	r, err := buildWidget(t, `
resources:
  widgets:
    deprecated: "Use netbox_gadget instead."
    attributes:
      legacy: { deprecated: false }          # suppress the spec flag
      old_ports: { skip: true }               # skipped attributes vanish
      description: { deprecated: "Use comments instead." }
      widget_count: { deprecated: true }      # read-only, default message
      items:
        nested:
          label: { optional: true, deprecated: "Use name instead." }
`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Deprecated != "Use netbox_gadget instead." {
		t.Errorf("resource deprecation = %q", r.Deprecated)
	}
	if a := r.Attr("legacy"); a.Deprecated != "" {
		t.Errorf("legacy: deprecated: false must suppress the spec flag, got %q", a.Deprecated)
	}
	if r.Attr("old_ports") != nil {
		t.Error("old_ports: skip must win over the spec deprecation")
	}
	if a := r.Attr("description"); a.Deprecated != "Use comments instead." {
		t.Errorf("description = %q", a.Deprecated)
	}
	if a := attrIn(r.ReadOnlyAttrs, "widget_count"); a.Deprecated != DefaultAttributeDeprecation {
		t.Errorf("widget_count = %q", a.Deprecated)
	}
	items := r.Attr("items")
	if l := attrIn(items.Nested, "label"); l.Deprecated != "Use name instead." || l.Required {
		t.Errorf("nested label = %+v", l)
	}
	if w := attrIn(items.Nested, "weight"); w.Deprecated != DefaultAttributeDeprecation {
		t.Errorf("nested weight keeps the spec deprecation, got %q", w.Deprecated)
	}
}

func TestDeprecationFromSpecOperation(t *testing.T) {
	r, err := buildWidget(t, "", func(d *openapi.Document) { d.Paths["/api/dcim/widgets/"].Get.Deprecated = true })
	if err != nil {
		t.Fatal(err)
	}
	if r.Deprecated != DefaultResourceDeprecation {
		t.Errorf("resource deprecation = %q", r.Deprecated)
	}
}

func TestDeprecationErrors(t *testing.T) {
	for name, tc := range map[string]struct{ yaml, want string }{
		"required attribute": {"resources:\n  widgets:\n    attributes:\n      name: { deprecated: \"x\" }\n", "required attribute cannot be deprecated"},
		"negative version":   {"resources:\n  widgets:\n    schema_version: -1\n", "schema_version must not be negative"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := buildWidget(t, tc.yaml, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
	var f overrides.File
	if err := yaml.Unmarshal([]byte("resources:\n  widgets:\n    deprecated: [a]\n"), &f); err == nil {
		t.Error("a list must not parse as a deprecation")
	}
	if err := yaml.Unmarshal([]byte("resources:\n  widgets:\n    deprecated: \"\"\n"), &f); err == nil {
		t.Error("an empty message must not parse")
	}
}

func TestSchemaVersionOverride(t *testing.T) {
	r, err := buildWidget(t, "resources:\n  widgets:\n    schema_version: 2\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.SchemaVersion != 2 {
		t.Fatalf("SchemaVersion = %d, want 2", r.SchemaVersion)
	}
}
