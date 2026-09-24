# Component names are unique per device; combine the device and the name.
data "netbox_device" "sw01" {
  name = "fra1-sw01"
}

data "netbox_cooling_outflow" "example" {
  filters = [
    { name = "device_id", value = data.netbox_device.sw01.id },
    { name = "name", value = "Outflow 1" },
  ]
}
