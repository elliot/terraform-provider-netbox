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
resource "netbox_vlan" "users" {
  name = "users"
  vid  = 100
}
resource "netbox_vlan" "voice" {
  name = "voice"
  vid  = 200
}
resource "netbox_interface" "gi1_0_1" {
  device_id        = netbox_device.sw01.id
  name             = "GigabitEthernet1/0/1"
  label            = "Gi1/0/1"
  type             = "1000base-t"
  enabled          = true
  mtu              = 1500
  mode             = "tagged"
  untagged_vlan_id = netbox_vlan.users.id
  tagged_vlan_ids  = [netbox_vlan.voice.id]
  poe_mode         = "pse"
  poe_type         = "type2-ieee802.3at"
  description      = "Desk 1-01"
}
resource "netbox_interface" "po1" {
  device_id = netbox_device.sw01.id
  name      = "Port-channel1"
  type      = "lag"
}
resource "netbox_interface" "te1_1_1" {
  device_id = netbox_device.sw01.id
  name      = "TenGigabitEthernet1/1/1"
  type      = "10gbase-x-sfpp"
  lag_id    = netbox_interface.po1.id
}
