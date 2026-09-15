data "netbox_device" "chassis" {
  name = "dc1-ucs-chassis01"
}
# Every device bay of one parent object.
data "netbox_device_bays" "all" {
  filters = [
    { name = "device_id", value = data.netbox_device.chassis.id },
  ]
}
output "device_bays_names" {
  value = data.netbox_device_bays.all.items[*].name
}
