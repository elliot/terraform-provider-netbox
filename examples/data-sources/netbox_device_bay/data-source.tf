data "netbox_device" "chassis" {
  name = "dc1-ucs-chassis01"
}
data "netbox_device_bay" "slot1" {
  filters = [
    { name = "device_id", value = data.netbox_device.chassis.id },
    { name = "name", value = "Slot 1" },
  ]
}
output "installed_blade" {
  value = data.netbox_device_bay.slot1.installed_device_id
}
