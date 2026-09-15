resource "netbox_manufacturer" "example" {
  name = "Example Networks"
  slug = "example-networks"
}

resource "netbox_device_type" "sw48" {
  manufacturer_id = netbox_manufacturer.example.id
  model           = "SW-48"
  slug            = "example-sw-48"
  u_height        = 1
}

resource "netbox_power_port_template" "psu1" {
  device_type_id = netbox_device_type.sw48.id
  name           = "PSU1"
  type           = "iec-60320-c14"
}

# power_port_id refers to a power port *template* of the same device type.
resource "netbox_power_outlet_template" "outlet_1" {
  device_type_id = netbox_device_type.sw48.id
  name           = "Outlet 1"
  label          = "A1"
  type           = "iec-60320-c13"
  power_port_id  = netbox_power_port_template.psu1.id
  feed_leg       = "A"
  color          = "ff9800"
}
