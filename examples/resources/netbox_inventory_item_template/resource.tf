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
resource "netbox_inventory_item_role" "psu" {
  name = "Power supply"
  slug = "power-supply"
}
resource "netbox_inventory_item_template" "psu" {
  count           = 2
  device_type_id  = netbox_device_type.c9300.id
  name            = "PSU ${count.index + 1}"
  label           = "PS${count.index + 1}"
  role_id         = netbox_inventory_item_role.psu.id
  manufacturer_id = netbox_manufacturer.cisco.id
  part_id         = "PWR-C1-715WAC"
}
