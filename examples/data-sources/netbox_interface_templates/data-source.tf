data "netbox_device_type" "example" {
  slug = "c9300-48p"
}
# Every interface template of one parent object.
data "netbox_interface_templates" "all" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.example.id },
  ]
}
output "interface_templates_names" {
  value = data.netbox_interface_templates.all.items[*].name
}
