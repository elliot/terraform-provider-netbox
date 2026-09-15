resource "netbox_manufacturer" "cisco" {
  name = "Cisco"
  slug = "cisco"
}
resource "netbox_device_type" "c9300" {
  manufacturer_id = netbox_manufacturer.cisco.id
  model           = "Catalyst 9300-48P"
  slug            = "c9300-48p"
  part_number     = "C9300-48P"
  u_height        = 1
}
resource "netbox_site" "dc1" {
  name = "DC1"
  slug = "dc1"
}
resource "netbox_device_role" "access_switch" {
  name  = "Access switch"
  slug  = "access-switch"
  color = "2196f3"
}
resource "netbox_device" "sw01" {
  name           = "dc1-access-sw01"
  device_type_id = netbox_device_type.c9300.id
  role_id        = netbox_device_role.access_switch.id
  site_id        = netbox_site.dc1.id
  status         = "active"
}
resource "netbox_module_type" "nm_8x" {
  manufacturer_id = netbox_manufacturer.cisco.id
  model           = "C9300-NM-8X"
  part_number     = "C9300-NM-8X"
  airflow         = "front-to-rear"
  weight          = 0.45
  weight_unit     = "kg"
  description     = "8x 10G SFP+ network module"
}
resource "netbox_module_bay" "nm1" {
  device_id = netbox_device.sw01.id
  name      = "Network Module 1"
  position  = "1"
}
resource "netbox_module" "nm1" {
  device_id      = netbox_device.sw01.id
  module_bay_id  = netbox_module_bay.nm1.id
  module_type_id = netbox_module_type.nm_8x.id
  status         = "active"
  serial         = "FOC2345X0CD"
  asset_tag      = "IT-000124"
  description    = "Uplink module"

  # Write-only flags honoured on create: replicate the module type's component
  # templates onto the device (default true) or adopt existing components.
  replicate_components = true
  adopt_components     = false
}
