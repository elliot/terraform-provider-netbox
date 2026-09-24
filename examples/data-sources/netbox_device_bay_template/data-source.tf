data "netbox_device_type" "ucs_5108" {
  slug = "ucs-5108"
}
data "netbox_device_bay_template" "example" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.ucs_5108.id },
    { name = "name", value = "Slot 1" },
  ]
}
