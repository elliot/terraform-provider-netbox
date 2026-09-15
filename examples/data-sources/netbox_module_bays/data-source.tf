data "netbox_device" "example" {
  name = "dc1-access-sw01"
}
# Every module bay of one parent object.
data "netbox_module_bays" "all" {
  filters = [
    { name = "device_id", value = data.netbox_device.example.id },
  ]
}
output "module_bays_names" {
  value = data.netbox_module_bays.all.items[*].name
}
