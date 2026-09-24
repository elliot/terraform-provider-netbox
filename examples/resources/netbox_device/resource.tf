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
resource "netbox_platform" "ios_xe" {
  name = "Cisco IOS-XE"
  slug = "cisco-ios-xe"
}
resource "netbox_device" "sw01" {
  name           = "dc1-access-sw01"
  device_type_id = netbox_device_type.c9300.id
  role_id        = netbox_device_role.access_switch.id
  site_id        = netbox_site.dc1.id
  platform_id    = netbox_platform.ios_xe.id
  status         = "active"
  serial         = "FCW2345L0AB"
  asset_tag      = "IT-000123"
  airflow        = "front-to-rear"
  description    = "Rack 1 top-of-rack access switch"
  comments       = "Managed by Terraform"
}
