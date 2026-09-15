data "netbox_device_type" "example" {
  slug = "c9300-48p"
}
# Every front port template of one parent object.
data "netbox_front_port_templates" "all" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.example.id },
  ]
}
output "front_port_templates_names" {
  value = data.netbox_front_port_templates.all.items[*].name
}
