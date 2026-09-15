data "netbox_device" "con01" {
  name = "dc1-con01"
}
data "netbox_console_server_port" "example" {
  filters = [
    { name = "device_id", value = data.netbox_device.con01.id },
    { name = "name", value = "port1" },
  ]
}
