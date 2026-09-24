data "netbox_wireless_lan_group" "campus" {
  slug = "campus"
}

resource "netbox_wireless_lan" "corp" {
  ssid     = "Corp-WiFi"
  group_id = data.netbox_wireless_lan_group.campus.id
}
