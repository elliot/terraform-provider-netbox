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
}

# cooling_intake_id refers to a cooling intake *template* of the same device type.
resource "netbox_cooling_outflow_template" "out1" {
  device_type_id    = netbox_device_type.sw48.id
  name              = "Outflow 1"
  label             = "Coolant return"
  type              = "uqdb"
  diameter          = 12.7
  diameter_unit     = "mm"
  cooling_intake_id = netbox_cooling_intake_template.in1.id
}
