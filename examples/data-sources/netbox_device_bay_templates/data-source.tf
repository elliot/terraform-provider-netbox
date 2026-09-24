data "netbox_device_type" "example" {
  slug = "c9300-48p"
}
# Every device bay template of one parent object.
data "netbox_device_bay_templates" "all" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.example.id },
  ]
}
output "device_bay_templates_names" {
  value = data.netbox_device_bay_templates.all.items[*].name
}
