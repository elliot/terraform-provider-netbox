data "netbox_device" "sw01" {
  name = "dc1-access-sw01"
}
data "netbox_virtual_device_context" "example" {
  filters = [
    { name = "device_id", value = data.netbox_device.sw01.id },
    { name = "name", value = "cust-a-vdc" },
  ]
}
