package manual_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/elliot/terraform-provider-netbox/internal/acctest"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
)

const availableIpAddressPrefixFixture = `resource "netbox_prefix" "test" {
  prefix      = "10.201.%[1]d.0/24"
  description = "{{.Name}}"
}
resource "netbox_tag" "test" {
  name = "{{.Name}}"
  slug = "{{.Name}}"
}
`

const availableIpAddressConfigBasic = availableIpAddressPrefixFixture + `
resource "netbox_available_ip_address" "test" {
  prefix_id   = netbox_prefix.test.id
  description = "{{.Name}}"
}
`

const availableIpAddressConfigUpdate = availableIpAddressPrefixFixture + `
resource "netbox_available_ip_address" "test" {
  prefix_id   = netbox_prefix.test.id
  description = "{{.Name}} updated"
  status      = "reserved"
  role        = "vip"
  dns_name    = "{{.Name}}.example.com"
  comments    = "allocated by terraform"
  tags        = [netbox_tag.test.slug]
}
`

func TestAccAvailableIpAddress_basic(t *testing.T) {
	name := acctest.RandName()
	o := octet(name)
	rn := "netbox_available_ip_address.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:             acctest.CheckDestroyed("netbox_available_ip_address", "/api/ipam/ip-addresses/"),
		Steps: []resource.TestStep{
			{
				Config: render(t, availableIpAddressConfigBasic, name, o),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "address", fmt.Sprintf("10.201.%d.1/24", o)),
					resource.TestCheckResourceAttr(rn, "status", "active"),
					resource.TestCheckResourceAttr(rn, "description", name),
					resource.TestCheckNoResourceAttr(rn, "vrf_id"),
					resource.TestCheckResourceAttrPair(rn, "prefix_id", "netbox_prefix.test", "id"),
				),
			},
			{
				Config: render(t, availableIpAddressConfigUpdate, name, o),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "address", fmt.Sprintf("10.201.%d.1/24", o)),
					resource.TestCheckResourceAttr(rn, "status", "reserved"),
					resource.TestCheckResourceAttr(rn, "role", "vip"),
					resource.TestCheckResourceAttr(rn, "dns_name", name+".example.com"),
					resource.TestCheckResourceAttr(rn, "tags.#", "1"),
				),
			},
			{
				ResourceName:            rn,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"prefix_id", "ip_range_id", "custom_fields"},
			},
			{
				Config: render(t, availableIpAddressConfigUpdate, name, o),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

const availableIpAddressRangeConfig = `resource "netbox_ip_range" "test" {
  start_address = "10.204.%[1]d.10/24"
  end_address   = "10.204.%[1]d.20/24"
  description   = "{{.Name}}"
}

resource "netbox_available_ip_address" "test" {
  ip_range_id = netbox_ip_range.test.id
  description = "{{.Name}}"
}
`

func TestAccAvailableIpAddress_ipRange(t *testing.T) {
	name := acctest.RandName()
	o := octet(name)
	rn := "netbox_available_ip_address.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:             acctest.CheckDestroyed("netbox_available_ip_address", "/api/ipam/ip-addresses/"),
		Steps: []resource.TestStep{
			{
				Config: render(t, availableIpAddressRangeConfig, name, o),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "address", fmt.Sprintf("10.204.%d.10/24", o)),
					resource.TestCheckResourceAttrPair(rn, "ip_range_id", "netbox_ip_range.test", "id"),
				),
			},
			{
				Config: render(t, availableIpAddressRangeConfig, name, o),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
