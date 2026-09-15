data "netbox_device_type" "example" {
  slug = "c9300-48p"
}
# Every console server port template of one parent object.
data "netbox_console_server_port_templates" "all" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.example.id },
  ]
}
output "console_server_port_templates_names" {
  value = data.netbox_console_server_port_templates.all.items[*].name
}
