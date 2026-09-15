package manual_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/elliot/terraform-provider-netbox/internal/acctest"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
)

// Every custom field type on one object: text, longtext, integer, decimal,
// boolean, date, url, json, select, multiselect, object and multiobject.
const customFieldsAllTypesConfig = `
resource "netbox_custom_field_choice_set" "test" {
  name          = "{{.Snake}}_tiers"
  extra_choices = [["bronze", "Bronze"], ["gold", "Gold"], ["silver", "Silver"]]
}

resource "netbox_custom_field" "text" {
  name         = "{{.Snake}}_text"
  type         = "text"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "longtext" {
  name         = "{{.Snake}}_longtext"
  type         = "longtext"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "integer" {
  name         = "{{.Snake}}_integer"
  type         = "integer"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "decimal" {
  name         = "{{.Snake}}_decimal"
  type         = "decimal"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "boolean" {
  name         = "{{.Snake}}_boolean"
  type         = "boolean"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "date" {
  name         = "{{.Snake}}_date"
  type         = "date"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "url" {
  name         = "{{.Snake}}_url"
  type         = "url"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "json" {
  name         = "{{.Snake}}_json"
  type         = "json"
  object_types = ["tenancy.tenant"]
}


resource "netbox_custom_field" "select" {
  name          = "{{.Snake}}_select"
  type          = "select"
  object_types  = ["tenancy.tenant"]
  choice_set_id = netbox_custom_field_choice_set.test.id
}

resource "netbox_custom_field" "multiselect" {
  name          = "{{.Snake}}_multiselect"
  type          = "multiselect"
  object_types  = ["tenancy.tenant"]
  choice_set_id = netbox_custom_field_choice_set.test.id
}

resource "netbox_custom_field" "object" {
  name                = "{{.Snake}}_object"
  type                = "object"
  object_types        = ["tenancy.tenant"]
  related_object_type = "dcim.site"
}

resource "netbox_custom_field" "multiobject" {
  name                = "{{.Snake}}_multiobject"
  type                = "multiobject"
  object_types        = ["tenancy.tenant"]
  related_object_type = "dcim.site"
}

resource "netbox_site" "a" {
  name = "{{.Name}}-a"
  slug = "{{.Name}}-a"
}

resource "netbox_site" "b" {
  name = "{{.Name}}-b"
  slug = "{{.Name}}-b"
}

resource "netbox_tenant" "test" {
  name = "{{.Name}}"
  slug = "{{.Name}}"
  custom_fields = {
    "{{.Snake}}_text"        = "hello"
    "{{.Snake}}_longtext"    = "line one\nline two"
    "{{.Snake}}_integer"     = 42
    "{{.Snake}}_decimal"     = 1.25
    "{{.Snake}}_boolean"     = true
    "{{.Snake}}_date"        = "2026-01-02"
    "{{.Snake}}_url"         = "https://example.com/x"
    "{{.Snake}}_json"        = { a = [1, 2], b = "x" }
    "{{.Snake}}_select"      = "gold"
    "{{.Snake}}_multiselect" = ["bronze", "gold"]
    "{{.Snake}}_object"      = netbox_site.a.id
    "{{.Snake}}_multiobject" = [netbox_site.a.id, netbox_site.b.id]
  }
  depends_on = [netbox_custom_field.text, netbox_custom_field.longtext, netbox_custom_field.integer, netbox_custom_field.decimal, netbox_custom_field.boolean, netbox_custom_field.date, netbox_custom_field.url, netbox_custom_field.json, netbox_custom_field.select, netbox_custom_field.multiselect, netbox_custom_field.object, netbox_custom_field.multiobject]
}
`

// Update: change values, drop one key (untracked afterwards), clear one to null.
const customFieldsAllTypesUpdate = `
resource "netbox_custom_field_choice_set" "test" {
  name          = "{{.Snake}}_tiers"
  extra_choices = [["bronze", "Bronze"], ["gold", "Gold"], ["silver", "Silver"]]
}

resource "netbox_custom_field" "text" {
  name         = "{{.Snake}}_text"
  type         = "text"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "longtext" {
  name         = "{{.Snake}}_longtext"
  type         = "longtext"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "integer" {
  name         = "{{.Snake}}_integer"
  type         = "integer"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "decimal" {
  name         = "{{.Snake}}_decimal"
  type         = "decimal"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "boolean" {
  name         = "{{.Snake}}_boolean"
  type         = "boolean"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "date" {
  name         = "{{.Snake}}_date"
  type         = "date"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "url" {
  name         = "{{.Snake}}_url"
  type         = "url"
  object_types = ["tenancy.tenant"]
}

resource "netbox_custom_field" "json" {
  name         = "{{.Snake}}_json"
  type         = "json"
  object_types = ["tenancy.tenant"]
}


resource "netbox_custom_field" "select" {
  name          = "{{.Snake}}_select"
  type          = "select"
  object_types  = ["tenancy.tenant"]
  choice_set_id = netbox_custom_field_choice_set.test.id
}

resource "netbox_custom_field" "multiselect" {
  name          = "{{.Snake}}_multiselect"
  type          = "multiselect"
  object_types  = ["tenancy.tenant"]
  choice_set_id = netbox_custom_field_choice_set.test.id
}

resource "netbox_custom_field" "object" {
  name                = "{{.Snake}}_object"
  type                = "object"
  object_types        = ["tenancy.tenant"]
  related_object_type = "dcim.site"
}

resource "netbox_custom_field" "multiobject" {
  name                = "{{.Snake}}_multiobject"
  type                = "multiobject"
  object_types        = ["tenancy.tenant"]
  related_object_type = "dcim.site"
}

resource "netbox_site" "a" {
  name = "{{.Name}}-a"
  slug = "{{.Name}}-a"
}

resource "netbox_site" "b" {
  name = "{{.Name}}-b"
  slug = "{{.Name}}-b"
}

resource "netbox_tenant" "test" {
  name = "{{.Name}}"
  slug = "{{.Name}}"
  custom_fields = {
    "{{.Snake}}_text"        = "changed"
    "{{.Snake}}_integer"     = 43
    "{{.Snake}}_boolean"     = false
    "{{.Snake}}_json"        = { a = [3], c = true }
    "{{.Snake}}_select"      = "silver"
    "{{.Snake}}_multiselect" = ["silver"]
    "{{.Snake}}_object"      = netbox_site.b.id
    "{{.Snake}}_multiobject" = [netbox_site.b.id]
    "{{.Snake}}_url"         = null
  }
  depends_on = [netbox_custom_field.text, netbox_custom_field.longtext, netbox_custom_field.integer, netbox_custom_field.decimal, netbox_custom_field.boolean, netbox_custom_field.date, netbox_custom_field.url, netbox_custom_field.json, netbox_custom_field.select, netbox_custom_field.multiselect, netbox_custom_field.object, netbox_custom_field.multiobject]
}

data "netbox_tenant" "test" {
  id = netbox_tenant.test.id
}
`

func TestAccCustomFields_allTypes(t *testing.T) {
	name := acctest.RandName()
	snake := snakeName(name)
	rn := "netbox_tenant.test"
	resource.Test(t, resource.TestCase{ // serial: custom fields are global on the shared demo
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:             acctest.CheckDestroyed("netbox_tenant", "/api/tenancy/tenants/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.Render(t, customFieldsAllTypesConfig, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "custom_fields."+snake+"_text", "hello"),
					resource.TestCheckResourceAttr(rn, "custom_fields."+snake+"_integer", "42"),
					resource.TestCheckResourceAttr(rn, "custom_fields."+snake+"_decimal", "1.25"),
					resource.TestCheckResourceAttr(rn, "custom_fields."+snake+"_boolean", "true"),
					resource.TestCheckResourceAttr(rn, "custom_fields."+snake+"_select", "gold"),
					resource.TestCheckResourceAttr(rn, "custom_fields."+snake+"_multiselect.#", "2"),
					resource.TestCheckResourceAttrPair(rn, "custom_fields."+snake+"_object", "netbox_site.a", "id"),
					resource.TestCheckResourceAttr(rn, "custom_fields."+snake+"_multiobject.#", "2"),
					resource.TestCheckResourceAttr(rn, "custom_fields."+snake+"_json.b", "x"),
				),
			},
			{
				Config: acctest.Render(t, customFieldsAllTypesUpdate, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "custom_fields."+snake+"_text", "changed"),
					resource.TestCheckResourceAttr(rn, "custom_fields."+snake+"_select", "silver"),
					resource.TestCheckResourceAttrPair(rn, "custom_fields."+snake+"_object", "netbox_site.b", "id"),
					resource.TestCheckNoResourceAttr(rn, "custom_fields."+snake+"_decimal"),
					// the data source exposes every field as JSON, including ones the resource no longer tracks
					resource.TestCheckResourceAttrWith("data.netbox_tenant.test", "custom_fields", func(v string) error {
						return checkJSONFields(v, map[string]any{snake + "_decimal": 1.25, snake + "_select": "silver", snake + "_text": "changed"})
					}),
				),
			},
			{
				ResourceName:            rn,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"custom_fields"},
			},
			{
				Config: acctest.Render(t, customFieldsAllTypesUpdate, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

const customFieldUnknownNameConfig = `
resource "netbox_tenant" "test" {
  name = "{{.Name}}"
  slug = "{{.Name}}"
  custom_fields = {
    does_not_exist_{{.Snake}} = "x"
  }
}
`

func TestAccCustomFields_unknownName(t *testing.T) {
	name := acctest.RandName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      acctest.Render(t, customFieldUnknownNameConfig, name),
			ExpectError: regexpMust(`Unknown custom field`),
		}},
	})
}

const customFieldValueConfig = `
resource "netbox_custom_field" "test" {
  name         = "{{.Snake}}_owner"
  type         = "text"
  object_types = ["dcim.site"]
}

resource "netbox_custom_field" "sites" {
  name                = "{{.Snake}}_peers"
  type                = "multiobject"
  object_types        = ["dcim.site"]
  related_object_type = "dcim.site"
}

resource "netbox_site" "test" {
  name = "{{.Name}}"
  slug = "{{.Name}}"
}

resource "netbox_site" "peer" {
  name = "{{.Name}}-peer"
  slug = "{{.Name}}-peer"
}

resource "netbox_custom_field_value" "owner" {
  object_type = "dcim.site"
  object_id   = netbox_site.test.id
  name        = netbox_custom_field.test.name
  value       = "%s"
}

resource "netbox_custom_field_value" "peers" {
  object_type = "dcim.site"
  object_id   = netbox_site.test.id
  name        = netbox_custom_field.sites.name
  value       = [netbox_site.peer.id]
}
`

func TestAccCustomFieldValue_basic(t *testing.T) {
	name := acctest.RandName()
	rn := "netbox_custom_field_value.owner"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: render(t, customFieldValueConfig, name, "team-a"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "value", "team-a"),
					resource.TestCheckResourceAttrPair("netbox_custom_field_value.peers", "value.0", "netbox_site.peer", "id"),
				),
			},
			{
				Config: render(t, customFieldValueConfig, name, "team-b"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate)},
				},
				Check: resource.TestCheckResourceAttr(rn, "value", "team-b"),
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraformState) (string, error) {
					return importID(s, rn)
				},
			},
			{
				Config: render(t, customFieldValueConfig, name, "team-b"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
