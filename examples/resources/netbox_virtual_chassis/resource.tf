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
resource "netbox_virtual_chassis" "stack1" {
  name        = "dc1-stack01"
  domain      = "stack01"
  description = "Two-member StackWise stack"
}
# Members join the chassis through the device; the master is elected by
# NetBox once members exist (or set master_id afterwards, see docs).
resource "netbox_device" "member" {
  count              = 2
  name               = "dc1-stack01-${count.index + 1}"
  device_type_id     = netbox_device_type.c9300.id
  role_id            = netbox_device_role.access_switch.id
  site_id            = netbox_site.dc1.id
  virtual_chassis_id = netbox_virtual_chassis.stack1.id
  vc_position        = count.index + 1
  vc_priority        = count.index == 0 ? 15 : 10
}
