data "netbox_device" "example" {
  name = "dc1-access-sw01"
}
# Every console port of one parent object.
data "netbox_console_ports" "all" {
  filters = [
    { name = "device_id", value = data.netbox_device.example.id },
  ]
}
output "console_ports_names" {
  value = data.netbox_console_ports.all.items[*].name
}
