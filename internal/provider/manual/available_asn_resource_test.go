package manual_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/elliot/terraform-provider-netbox/internal/acctest"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
)

const availableAsnFixture = `resource "netbox_rir" "test" {
  name       = "{{.Name}}"
  slug       = "{{.Name}}"
  is_private = true
}
resource "netbox_asn_range" "test" {
  name   = "{{.Name}}"
  slug   = "{{.Name}}"
  rir_id = netbox_rir.test.id
  start  = %[1]d
  end    = %[2]d
}
`

const availableAsnConfigBasic = availableAsnFixture + `
resource "netbox_available_asn" "test" {
  asn_range_id = netbox_asn_range.test.id
  description  = "{{.Name}}"
}
`

const availableAsnConfigUpdate = availableAsnFixture + `
resource "netbox_available_asn" "test" {
  asn_range_id = netbox_asn_range.test.id
  description  = "{{.Name}} updated"
  comments     = "allocated by terraform"
}
`

func TestAccAvailableAsn_basic(t *testing.T) {
	name := acctest.RandName()
	// A private 32-bit ASN block unlikely to collide with other test runs.
	start := 4200000000 + (hash(name)%900000)*100
	end := start + 9
	rn := "netbox_available_asn.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:             acctest.CheckDestroyed("netbox_available_asn", "/api/ipam/asns/"),
		Steps: []resource.TestStep{
			{
				Config: render(t, availableAsnConfigBasic, name, start, end),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "asn", fmt.Sprint(start)),
					resource.TestCheckResourceAttrPair(rn, "rir_id", "netbox_rir.test", "id"),
					resource.TestCheckResourceAttrPair(rn, "asn_range_id", "netbox_asn_range.test", "id"),
				),
			},
			{
				Config: render(t, availableAsnConfigUpdate, name, start, end),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "asn", fmt.Sprint(start)),
					resource.TestCheckResourceAttr(rn, "description", name+" updated"),
				),
			},
			{
				ResourceName:            rn,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"asn_range_id", "custom_fields"},
			},
			{
				Config: render(t, availableAsnConfigUpdate, name, start, end),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
