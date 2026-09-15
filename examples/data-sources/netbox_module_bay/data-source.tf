data "netbox_device" "sw01" {
  name = "dc1-access-sw01"
}
data "netbox_module_bay" "example" {
  filters = [
    { name = "device_id", value = data.netbox_device.sw01.id },
    { name = "name", value = "Network Module 1" },
  ]
}
