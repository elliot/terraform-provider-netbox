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
resource "netbox_module_bay_template" "nm1" {
  device_type_id = netbox_device_type.c9300.id
  name           = "Network Module 1"
  label          = "NM1"
  position       = "1"
}
