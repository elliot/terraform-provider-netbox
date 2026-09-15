data "netbox_device" "pp01" {
  name = "dc1-r01-pp01"
}
data "netbox_rear_port" "example" {
  filters = [
    { name = "device_id", value = data.netbox_device.pp01.id },
    { name = "name", value = "Trunk 1" },
  ]
}
