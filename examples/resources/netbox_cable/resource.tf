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
resource "netbox_device" "sw02" {
  name           = "dc1-access-sw02"
  device_type_id = netbox_device_type.c9300.id
  role_id        = netbox_device_role.access_switch.id
  site_id        = netbox_site.dc1.id
}
resource "netbox_interface" "sw01_te1_1_1" {
  device_id = netbox_device.sw01.id
  name      = "TenGigabitEthernet1/1/1"
  type      = "10gbase-x-sfpp"
}
resource "netbox_interface" "sw02_te1_1_1" {
  device_id = netbox_device.sw02.id
  name      = "TenGigabitEthernet1/1/1"
  type      = "10gbase-x-sfpp"
}
resource "netbox_cable" "sw01_sw02" {
  a_terminations = [{ object_type = "dcim.interface", object_id = netbox_interface.sw01_te1_1_1.id }]
  b_terminations = [{ object_type = "dcim.interface", object_id = netbox_interface.sw02_te1_1_1.id }]
  type           = "smf-os2"
  status         = "connected"
  label          = "DC1-R01-0001"
  color          = "ffeb3b"
  length         = 3
  length_unit    = "m"
  description    = "Peer link"
}
