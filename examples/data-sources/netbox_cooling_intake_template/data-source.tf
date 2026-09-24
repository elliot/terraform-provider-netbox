# Template names are unique per device type (or module type).
data "netbox_device_type" "sw48" {
  slug = "example-sw-48"
}

data "netbox_cooling_intake_template" "example" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.sw48.id },
    { name = "name", value = "Intake 1" },
  ]
}
