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
# Interface templates are instantiated on every device created from the type.
resource "netbox_interface_template" "access" {
  count          = 48
  device_type_id = netbox_device_type.c9300.id
  name           = "GigabitEthernet1/0/${count.index + 1}"
  label          = "Gi1/0/${count.index + 1}"
  type           = "1000base-t"
  poe_mode       = "pse"
  poe_type       = "type2-ieee802.3at"
}
resource "netbox_interface_template" "mgmt" {
  device_type_id = netbox_device_type.c9300.id
  name           = "GigabitEthernet0/0"
  type           = "1000base-t"
  mgmt_only      = true
  description    = "Out-of-band management"
}
# Templates can also belong to a module type; {module} is replaced by the
# module bay position when the module is installed.
resource "netbox_module_type" "nm_8x" {
  manufacturer_id = netbox_manufacturer.cisco.id
  model           = "C9300-NM-8X"
}
resource "netbox_interface_template" "uplinks" {
  count          = 8
  module_type_id = netbox_module_type.nm_8x.id
  name           = "TenGigabitEthernet1/{module}/${count.index + 1}"
  type           = "10gbase-x-sfpp"
}
