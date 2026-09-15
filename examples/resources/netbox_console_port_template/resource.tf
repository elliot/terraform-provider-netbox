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
resource "netbox_console_port_template" "console" {
  device_type_id = netbox_device_type.c9300.id
  name           = "Console"
  label          = "CON"
  type           = "rj-45"
}
resource "netbox_console_port_template" "usb" {
  device_type_id = netbox_device_type.c9300.id
  name           = "USB Console"
  type           = "usb-mini-b"
}
