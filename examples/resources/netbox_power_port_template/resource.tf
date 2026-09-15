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

# Component templates belong to either a device type or a module type; every
# device instantiated from the type gets a matching power port.
resource "netbox_power_port_template" "psu1" {
  device_type_id = netbox_device_type.sw48.id
  name           = "PSU1"
  label          = "Power supply 1"
  type           = "iec-60320-c14"
  maximum_draw   = 500
  allocated_draw = 350
}
