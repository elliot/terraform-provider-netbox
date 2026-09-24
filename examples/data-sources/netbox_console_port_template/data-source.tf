data "netbox_device_type" "c9300" {
  slug = "c9300-48p"
}
data "netbox_console_port_template" "example" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.c9300.id },
    { name = "name", value = "Console" },
  ]
}
