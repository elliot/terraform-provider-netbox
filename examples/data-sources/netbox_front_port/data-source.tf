data "netbox_device" "pp01" {
  name = "dc1-r01-pp01"
}
data "netbox_front_port" "example" {
  filters = [
    { name = "device_id", value = data.netbox_device.pp01.id },
    { name = "name", value = "LC 1" },
  ]
}
output "front_port_mappings" {
  value = data.netbox_front_port.example.rear_ports
}
