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
resource "netbox_inventory_item_role" "psu" {
  name = "Power supply"
  slug = "power-supply"
}
resource "netbox_inventory_item" "psu1" {
  device_id       = netbox_device.sw01.id
  name            = "PSU 1"
  label           = "PS1"
  role_id         = netbox_inventory_item_role.psu.id
  manufacturer_id = netbox_manufacturer.cisco.id
  part_id         = "PWR-C1-715WAC"
  serial          = "DTN2345A0EF"
  status          = "active"
  description     = "715W AC power supply"
}
# Inventory items can be nested and associated with a device component.
resource "netbox_interface" "te1_1_1" {
  device_id = netbox_device.sw01.id
  name      = "TenGigabitEthernet1/1/1"
  type      = "10gbase-x-sfpp"
}
resource "netbox_inventory_item" "sfp" {
  device_id      = netbox_device.sw01.id
  name           = "SFP-10G-SR in Te1/1/1"
  part_id        = "SFP-10G-SR"
  component_type = "dcim.interface"
  component_id   = netbox_interface.te1_1_1.id
  discovered     = true
}
