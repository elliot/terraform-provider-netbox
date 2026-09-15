resource "netbox_wireless_lan_group" "campus" {
  name = "Campus"
  slug = "campus"
}

resource "netbox_vlan" "guest" {
  name = "guest-wifi"
  vid  = 200
}

resource "netbox_wireless_lan" "guest" {
  ssid        = "Guest-WiFi"
  description = "Captive-portal guest network"
  group_id    = netbox_wireless_lan_group.campus.id
  vlan_id     = netbox_vlan.guest.id
  status      = "active"
  auth_type   = "wpa-personal"
  auth_cipher = "aes"
  auth_psk    = var.guest_psk
}
