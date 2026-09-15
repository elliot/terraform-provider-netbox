data "netbox_device" "example" {
  name = "dc1-access-sw01"
}
# Every virtual device context of one parent object.
data "netbox_virtual_device_contexts" "all" {
  filters = [
    { name = "device_id", value = data.netbox_device.example.id },
  ]
}
output "virtual_device_contexts_names" {
  value = data.netbox_virtual_device_contexts.all.items[*].name
}
