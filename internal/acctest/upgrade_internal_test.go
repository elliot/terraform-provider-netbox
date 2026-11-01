package acctest

import (
	"reflect"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
)

func TestIncompatibilities(t *testing.T) {
	schema := &tfjson.ProviderSchema{
		ResourceSchemas: map[string]*tfjson.Schema{
			"netbox_site": {Block: &tfjson.SchemaBlock{Attributes: map[string]*tfjson.SchemaAttribute{
				"name": {}, "slug": {}, "tags": {},
			}}},
			"netbox_cable": {Block: &tfjson.SchemaBlock{Attributes: map[string]*tfjson.SchemaAttribute{
				"a_terminations": {AttributeNestedType: &tfjson.SchemaNestedAttributeType{
					Attributes: map[string]*tfjson.SchemaAttribute{"object_type": {}, "object_id": {}},
				}},
			}}},
		},
		DataSourceSchemas: map[string]*tfjson.Schema{
			"netbox_site": {Block: &tfjson.SchemaBlock{Attributes: map[string]*tfjson.SchemaAttribute{"id": {}}}},
		},
	}
	tests := []struct {
		name string
		cfg  string
		want []string
	}{
		{
			name: "compatible",
			cfg: `provider "netbox" {}
resource "netbox_site" "test" {
  name       = "x"
  slug       = "x"
  depends_on = []
}
data "netbox_site" "by_id" {
  id = netbox_site.test.id
}
`,
		},
		{
			name: "new attribute",
			cfg: `resource "netbox_site" "test" {
  name     = "x"
  owner_id = 1
}`,
			want: []string{"netbox_site.owner_id is unknown"},
		},
		{
			name: "new resource and data source",
			cfg: `resource "netbox_owner" "test" {
  name = "x"
}
data "netbox_owners" "all" {}`,
			want: []string{"resource netbox_owner is unknown", "data netbox_owners is unknown"},
		},
		{
			name: "new nested attribute",
			cfg: `resource "netbox_cable" "test" {
  a_terminations = [{ object_type = "dcim.interface", object_id = 1, position = 2 }]
}`,
			want: []string{"netbox_cable.a_terminations.position is unknown"},
		},
		{
			name: "nested block",
			cfg: `resource "netbox_site" "test" {
  name = "x"
  timeouts {
    create = "1m"
  }
}`,
			want: []string{"netbox_site.timeouts is unknown"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := incompatibilities(tt.cfg, schema)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
	if _, err := incompatibilities(`resource "x" {`, schema); err == nil {
		t.Error("expected a parse error")
	}
}
