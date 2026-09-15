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

resource "netbox_cooling_intake_template" "in1" {
  device_type_id = netbox_device_type.sw48.id
  name           = "Intake 1"
  label          = "Coolant in"
  type           = "uqd"
  diameter       = 12.7
  diameter_unit  = "mm"
  max_flow       = 4.5
  max_flow_unit  = "lpm"
}
