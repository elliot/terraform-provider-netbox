data "netbox_device_type" "example" {
  slug = "c9300-48p"
}
# Every inventory item template of one parent object.
data "netbox_inventory_item_templates" "all" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.example.id },
  ]
}
output "inventory_item_templates_names" {
  value = data.netbox_inventory_item_templates.all.items[*].name
}
