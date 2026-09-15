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

resource "netbox_cooling_outflow" "out1" {
  device_id = netbox_device.sw01.id
  name      = "Outflow 1"
}

# A liquid-cooling intake on the device, paired with its return (outflow).
resource "netbox_cooling_intake" "in1" {
  device_id          = netbox_device.sw01.id
  name               = "Intake 1"
  label              = "Coolant in"
  type               = "uqd"
  diameter           = 12.7
  diameter_unit      = "mm"
  max_flow           = 4.5
  max_flow_unit      = "lpm"
  cooling_outflow_id = netbox_cooling_outflow.out1.id
  description        = "Cold plate loop supply"
}
