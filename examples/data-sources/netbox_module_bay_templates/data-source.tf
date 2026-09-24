data "netbox_device_type" "example" {
  slug = "c9300-48p"
}
# Every module bay template of one parent object.
data "netbox_module_bay_templates" "all" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.example.id },
  ]
}
output "module_bay_templates_names" {
  value = data.netbox_module_bay_templates.all.items[*].name
}
