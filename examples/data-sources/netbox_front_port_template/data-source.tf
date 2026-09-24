data "netbox_device_type" "pp" {
  slug = "pp-24-lc"
}
data "netbox_front_port_template" "example" {
  filters = [
    { name = "device_type_id", value = data.netbox_device_type.pp.id },
    { name = "name", value = "LC 1" },
  ]
}
