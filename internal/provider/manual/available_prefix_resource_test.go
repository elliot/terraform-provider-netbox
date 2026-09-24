package manual_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/elliot/terraform-provider-netbox/internal/acctest"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
)

const availablePrefixFixture = `resource "netbox_prefix" "parent" {
  prefix      = "10.203.%[1]d.0/24"
  status      = "container"
  description = "{{.Name}} parent"
}
`

const availablePrefixConfigBasic = availablePrefixFixture + `
resource "netbox_available_prefix" "test" {
  parent_prefix_id = netbox_prefix.parent.id
  prefix_length    = 28
  description      = "{{.Name}}"
}
`

const availablePrefixConfigUpdate = availablePrefixFixture + `
resource "netbox_available_prefix" "test" {
  parent_prefix_id = netbox_prefix.parent.id
  prefix_length    = 28
  description      = "{{.Name}} updated"
  status           = "reserved"
  is_pool          = true
  mark_utilized    = true
  comments         = "allocated by terraform"
}
`

func TestAccAvailablePrefix_basic(t *testing.T) {
	name := acctest.RandName()
	o := octet(name)
	rn := "netbox_available_prefix.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:             acctest.CheckDestroyed("netbox_available_prefix", "/api/ipam/prefixes/"),
		Steps: []resource.TestStep{
			{
				Config: render(t, availablePrefixConfigBasic, name, o),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "prefix", fmt.Sprintf("10.203.%d.0/28", o)),
					resource.TestCheckResourceAttr(rn, "status", "active"),
					resource.TestCheckResourceAttr(rn, "is_pool", "false"),
					resource.TestCheckNoResourceAttr(rn, "vrf_id"),
					resource.TestCheckResourceAttrPair(rn, "parent_prefix_id", "netbox_prefix.parent", "id"),
				),
			},
			{
				Config: render(t, availablePrefixConfigUpdate, name, o),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "prefix", fmt.Sprintf("10.203.%d.0/28", o)),
					resource.TestCheckResourceAttr(rn, "status", "reserved"),
					resource.TestCheckResourceAttr(rn, "is_pool", "true"),
					resource.TestCheckResourceAttr(rn, "mark_utilized", "true"),
				),
			},
			{
				ResourceName:            rn,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"parent_prefix_id", "prefix_length", "custom_fields"},
			},
			{
				Config: render(t, availablePrefixConfigUpdate, name, o),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
