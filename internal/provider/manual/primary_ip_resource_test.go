package manual_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/elliot/terraform-provider-netbox/internal/acctest"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
)

const devicePrimaryIpFixture = `resource "netbox_manufacturer" "test" {
  name = "{{.Name}}"
  slug = "{{.Name}}"
}
resource "netbox_device_type" "test" {
  manufacturer_id = netbox_manufacturer.test.id
  model           = "{{.Name}}"
  slug            = "{{.Name}}"
}
resource "netbox_device_role" "test" {
  name = "{{.Name}}"
  slug = "{{.Name}}"
}
resource "netbox_site" "test" {
  name = "{{.Name}}"
  slug = "{{.Name}}"
}
resource "netbox_device" "test" {
  name           = "{{.Name}}"
  device_type_id = netbox_device_type.test.id
  role_id        = netbox_device_role.test.id
  site_id        = netbox_site.test.id
  lifecycle {
    ignore_changes = [primary_ip4_id, primary_ip6_id]
  }
}
resource "netbox_interface" "test" {
  device_id = netbox_device.test.id
  name      = "eth0"
  type      = "1000base-t"
}
resource "netbox_ip_address" "first" {
  address              = "10.205.%[1]d.1/24"
  description          = "{{.Name}}"
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.test.id
}
resource "netbox_ip_address" "second" {
  address              = "10.205.%[1]d.2/24"
  description          = "{{.Name}}"
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.test.id
}
`

const devicePrimaryIpConfigBasic = devicePrimaryIpFixture + `
resource "netbox_device_primary_ip" "test" {
  device_id     = netbox_device.test.id
  ip_address_id = netbox_ip_address.first.id
}
`

const devicePrimaryIpConfigUpdate = devicePrimaryIpFixture + `
resource "netbox_device_primary_ip" "test" {
  device_id     = netbox_device.test.id
  ip_address_id = netbox_ip_address.second.id
}
`

func TestAccDevicePrimaryIp_basic(t *testing.T) {
	name := acctest.RandName()
	o := octet(name)
	rn := "netbox_device_primary_ip.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:             acctest.CheckDestroyed("netbox_device_primary_ip", "/api/dcim/devices/"),
		Steps: []resource.TestStep{
			{
				Config: render(t, devicePrimaryIpConfigBasic, name, o),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(rn, "id", "netbox_device.test", "id"),
					resource.TestCheckResourceAttrPair(rn, "device_id", "netbox_device.test", "id"),
					resource.TestCheckResourceAttrPair(rn, "ip_address_id", "netbox_ip_address.first", "id"),
					resource.TestCheckResourceAttr(rn, "ip_address_version", "4"),
				),
			},
			{
				Config: render(t, devicePrimaryIpConfigUpdate, name, o),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(rn, "ip_address_id", "netbox_ip_address.second", "id"),
					resource.TestCheckResourceAttr(rn, "ip_address_version", "4"),
				),
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: render(t, devicePrimaryIpConfigUpdate, name, o),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

const virtualMachinePrimaryIpFixture = `resource "netbox_cluster_type" "test" {
  name = "{{.Name}}"
  slug = "{{.Name}}"
}
resource "netbox_cluster" "test" {
  name    = "{{.Name}}"
  type_id = netbox_cluster_type.test.id
}
resource "netbox_virtual_machine" "test" {
  name       = "{{.Name}}"
  cluster_id = netbox_cluster.test.id
  lifecycle {
    ignore_changes = [primary_ip4_id, primary_ip6_id]
  }
}
resource "netbox_vm_interface" "test" {
  virtual_machine_id = netbox_virtual_machine.test.id
  name               = "eth0"
}
resource "netbox_ip_address" "first" {
  address              = "10.206.%[1]d.1/24"
  description          = "{{.Name}}"
  assigned_object_type = "virtualization.vminterface"
  assigned_object_id   = netbox_vm_interface.test.id
}
resource "netbox_ip_address" "second" {
  address              = "10.206.%[1]d.2/24"
  description          = "{{.Name}}"
  assigned_object_type = "virtualization.vminterface"
  assigned_object_id   = netbox_vm_interface.test.id
}
`

const virtualMachinePrimaryIpConfigBasic = virtualMachinePrimaryIpFixture + `
resource "netbox_virtual_machine_primary_ip" "test" {
  virtual_machine_id = netbox_virtual_machine.test.id
  ip_address_id      = netbox_ip_address.first.id
}
`

const virtualMachinePrimaryIpConfigUpdate = virtualMachinePrimaryIpFixture + `
resource "netbox_virtual_machine_primary_ip" "test" {
  virtual_machine_id = netbox_virtual_machine.test.id
  ip_address_id      = netbox_ip_address.second.id
  ip_address_version = 4
}
`

func TestAccVirtualMachinePrimaryIp_basic(t *testing.T) {
	name := acctest.RandName()
	o := octet(name)
	rn := "netbox_virtual_machine_primary_ip.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:             acctest.CheckDestroyed("netbox_virtual_machine_primary_ip", "/api/virtualization/virtual-machines/"),
		Steps: []resource.TestStep{
			{
				Config: render(t, virtualMachinePrimaryIpConfigBasic, name, o),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(rn, "id", "netbox_virtual_machine.test", "id"),
					resource.TestCheckResourceAttrPair(rn, "virtual_machine_id", "netbox_virtual_machine.test", "id"),
					resource.TestCheckResourceAttrPair(rn, "ip_address_id", "netbox_ip_address.first", "id"),
					resource.TestCheckResourceAttr(rn, "ip_address_version", "4"),
				),
			},
			{
				Config: render(t, virtualMachinePrimaryIpConfigUpdate, name, o),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(rn, "ip_address_id", "netbox_ip_address.second", "id"),
					resource.TestCheckResourceAttr(rn, "ip_address_version", "4"),
				),
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: render(t, virtualMachinePrimaryIpConfigUpdate, name, o),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
