data "netbox_device" "example" {
  name = "dc1-access-sw01"
}
# Every module of one parent object.
data "netbox_modules" "all" {
  filters = [
    { name = "device_id", value = data.netbox_device.example.id },
  ]
}
output "modules_names" {
  value = data.netbox_modules.all.items[*].display
}
