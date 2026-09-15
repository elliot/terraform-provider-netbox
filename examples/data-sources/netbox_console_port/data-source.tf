data "netbox_device" "sw01" {
  name = "dc1-access-sw01"
}
data "netbox_console_port" "example" {
  filters = [
    { name = "device_id", value = data.netbox_device.sw01.id },
    { name = "name", value = "Console" },
  ]
}
