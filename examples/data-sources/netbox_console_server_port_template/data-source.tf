data "netbox_device_type" "om2248" {
  slug = "om2248"
}
data "netbox_console_server_port_template" "example" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.om2248.id },
    { name = "name", value = "port1" },
  ]
}
