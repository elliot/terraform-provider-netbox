resource "netbox_manufacturer" "cisco" {
  name = "Cisco"
  slug = "cisco"
}
# A device bay only exists on a device whose type is a "parent" and can hold
# a "child" device (u_height 0) installed in it.
resource "netbox_device_type" "chassis" {
  manufacturer_id = netbox_manufacturer.cisco.id
  model           = "UCS 5108 Chassis"
  slug            = "ucs-5108"
  u_height        = 6
  subdevice_role  = "parent"
}
resource "netbox_device_type" "blade" {
  manufacturer_id = netbox_manufacturer.cisco.id
  model           = "UCS B200 M6"
  slug            = "ucs-b200-m6"
  u_height        = 0
  subdevice_role  = "child"
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
resource "netbox_device" "chassis" {
  name           = "dc1-ucs-chassis01"
  device_type_id = netbox_device_type.chassis.id
  role_id        = netbox_device_role.access_switch.id
  site_id        = netbox_site.dc1.id
}
resource "netbox_device" "blade1" {
  name           = "dc1-ucs-blade01"
  device_type_id = netbox_device_type.blade.id
  role_id        = netbox_device_role.access_switch.id
  site_id        = netbox_site.dc1.id
}
resource "netbox_device_bay" "slot1" {
  device_id           = netbox_device.chassis.id
  name                = "Slot 1"
  label               = "1"
  installed_device_id = netbox_device.blade1.id
  description         = "Half-width blade slot"
}
