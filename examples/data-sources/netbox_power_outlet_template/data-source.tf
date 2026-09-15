# Template names are unique per device type (or module type).
data "netbox_device_type" "sw48" {
  slug = "example-sw-48"
}

data "netbox_power_outlet_template" "example" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.sw48.id },
    { name = "name", value = "Outlet 1" },
  ]
}
