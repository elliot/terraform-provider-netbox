data "netbox_device" "example" {
  name = "dc1-access-sw01"
}
# Every front port of one parent object.
data "netbox_front_ports" "all" {
  filters = [
    { name = "device_id", value = data.netbox_device.example.id },
  ]
}
output "front_ports_names" {
  value = data.netbox_front_ports.all.items[*].name
}
