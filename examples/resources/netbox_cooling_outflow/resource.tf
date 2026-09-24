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

resource "netbox_device_role" "access_switch" {
  name  = "Access switch"
  slug  = "access-switch"
  color = "2196f3"
}

resource "netbox_site" "fra1" {
  name = "FRA1"
  slug = "fra1"
}

resource "netbox_device" "sw01" {
  name           = "fra1-sw01"
  device_type_id = netbox_device_type.sw48.id
  role_id        = netbox_device_role.access_switch.id
  site_id        = netbox_site.fra1.id
}

resource "netbox_cooling_intake" "in1" {
  device_id = netbox_device.sw01.id
  name      = "Intake 1"
}

resource "netbox_cooling_outflow" "out1" {
  device_id         = netbox_device.sw01.id
  name              = "Outflow 1"
  label             = "Coolant return"
  type              = "uqdb"
  diameter          = 12.7
  diameter_unit     = "mm"
  cooling_intake_id = netbox_cooling_intake.in1.id
  description       = "Cold plate loop return"
}
