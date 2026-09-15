resource "netbox_manufacturer" "cisco" {
  name = "Cisco"
  slug = "cisco"
}
resource "netbox_device_type" "chassis" {
  manufacturer_id = netbox_manufacturer.cisco.id
  model           = "UCS 5108 Chassis"
  slug            = "ucs-5108"
  u_height        = 6
  subdevice_role  = "parent"
}
# Every device created from this type gets bays "Slot 1" .. "Slot 8".
resource "netbox_device_bay_template" "slots" {
  count          = 8
  device_type_id = netbox_device_type.chassis.id
  name           = "Slot ${count.index + 1}"
  label          = tostring(count.index + 1)
}
