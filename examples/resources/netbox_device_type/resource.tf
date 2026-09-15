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
  is_full_depth   = false
  airflow         = "front-to-rear"
  weight          = 7.8
  weight_unit     = "kg"
  description     = "48-port PoE+ access switch"
}
