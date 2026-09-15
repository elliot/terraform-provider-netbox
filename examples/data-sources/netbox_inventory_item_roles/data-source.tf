# Every inventory item role of one parent object.
data "netbox_inventory_item_roles" "all" {
  filters = [
    { name = "q", value = "power" },
  ]
}
output "inventory_item_roles_names" {
  value = data.netbox_inventory_item_roles.all.items[*].name
}
