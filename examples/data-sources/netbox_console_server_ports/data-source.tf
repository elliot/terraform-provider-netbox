data "netbox_device" "example" {
  name = "dc1-access-sw01"
}
# Every console server port of one parent object.
data "netbox_console_server_ports" "all" {
  filters = [
    { name = "device_id", value = data.netbox_device.example.id },
  ]
}
output "console_server_ports_names" {
  value = data.netbox_console_server_ports.all.items[*].name
}
