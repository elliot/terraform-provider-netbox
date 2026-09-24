data "netbox_device" "sw01" {
  name = "dc1-access-sw01"
}

data "netbox_module_bay" "nm1" {
  filters = [
    { name = "device_id", value = data.netbox_device.sw01.id },
    { name = "name", value = "Network Module 1" },
  ]
}

# Modules have no name; look them up by device and bay.
data "netbox_module" "nm1" {
  filters = [
    { name = "device_id", value = data.netbox_device.sw01.id },
    { name = "module_bay_id", value = data.netbox_module_bay.nm1.id },
  ]
}
