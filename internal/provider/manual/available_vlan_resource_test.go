package manual_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/elliot/terraform-provider-netbox/internal/acctest"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
)

const availableVlanFixture = `resource "netbox_vlan_group" "test" {
  name       = "{{.Name}}"
  slug       = "{{.Name}}"
  vid_ranges = [[100, 199]]
}
`

const availableVlanConfigBasic = availableVlanFixture + `
resource "netbox_available_vlan" "test" {
  vlan_group_id = netbox_vlan_group.test.id
  name          = "{{.Name}}"
}
`

const availableVlanConfigUpdate = availableVlanFixture + `
resource "netbox_available_vlan" "test" {
  vlan_group_id = netbox_vlan_group.test.id
  name          = "{{.Name}} updated"
  status        = "reserved"
  description   = "allocated by terraform"
}
`

func TestAccAvailableVlan_basic(t *testing.T) {
	name := acctest.RandName()
	rn := "netbox_available_vlan.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:             acctest.CheckDestroyed("netbox_available_vlan", "/api/ipam/vlans/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.Render(t, availableVlanConfigBasic, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "vid", "100"),
					resource.TestCheckResourceAttr(rn, "status", "active"),
					resource.TestCheckResourceAttrPair(rn, "vlan_group_id", "netbox_vlan_group.test", "id"),
				),
			},
			{
				Config: acctest.Render(t, availableVlanConfigUpdate, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "vid", "100"),
					resource.TestCheckResourceAttr(rn, "name", name+" updated"),
					resource.TestCheckResourceAttr(rn, "status", "reserved"),
				),
			},
			{
				ResourceName:            rn,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"custom_fields"},
			},
			{
				Config: acctest.Render(t, availableVlanConfigUpdate, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
