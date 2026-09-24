data "netbox_device" "example" {
  name = "dc1-access-sw01"
}
# Every rear port of one parent object.
data "netbox_rear_ports" "all" {
  filters = [
    { name = "device_id", value = data.netbox_device.example.id },
  ]
}
output "rear_ports_names" {
  value = data.netbox_rear_ports.all.items[*].name
}
