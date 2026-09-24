# Component names are unique per device; combine the device and the name.
data "netbox_device" "sw01" {
  name = "fra1-sw01"
}

data "netbox_power_port" "example" {
  filters = [
    { name = "device_id", value = data.netbox_device.sw01.id },
    { name = "name", value = "PSU1" },
  ]
}
