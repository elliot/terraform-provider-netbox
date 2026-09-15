data "netbox_device" "example" {
  name = "dc1-access-sw01"
}
# Every inventory item of one parent object.
data "netbox_inventory_items" "all" {
  filters = [
    { name = "device_id", value = data.netbox_device.example.id },
  ]
}
output "inventory_items_names" {
  value = data.netbox_inventory_items.all.items[*].name
}
