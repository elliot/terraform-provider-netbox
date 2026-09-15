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

resource "netbox_power_port" "psu1" {
  device_id = netbox_device.sw01.id
  name      = "PSU1"
  type      = "iec-60320-c14"
}

# An outlet fed by PSU1 (typical for a PDU or an inline power device).
resource "netbox_power_outlet" "outlet_1" {
  device_id     = netbox_device.sw01.id
  name          = "Outlet 1"
  label         = "A1"
  type          = "iec-60320-c13"
  status        = "enabled"
  power_port_id = netbox_power_port.psu1.id
  feed_leg      = "A"
  color         = "ff9800"
  description   = "Feeds the top-of-rack switch"
}
