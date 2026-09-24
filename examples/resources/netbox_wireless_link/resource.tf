# A wireless link joins two wireless interfaces (type ieee802.11*) on
# different devices.
resource "netbox_interface" "tower_a" {
  device_id = netbox_device.tower_a.id
  name      = "radio0"
  type      = "ieee802.11ax"
  rf_role   = "ap"
}

resource "netbox_interface" "tower_b" {
  device_id = netbox_device.tower_b.id
  name      = "radio0"
  type      = "ieee802.11ax"
  rf_role   = "station"
}

resource "netbox_wireless_link" "backhaul" {
  interface_a_id = netbox_interface.tower_a.id
  interface_b_id = netbox_interface.tower_b.id
  ssid           = "backhaul-a-b"
  status         = "connected"
  auth_type      = "wpa-enterprise"
  auth_cipher    = "aes"
  distance       = 1.8
  distance_unit  = "km"
  description    = "Point-to-point backhaul between tower A and tower B"
}
