resource "netbox_wireless_lan_group" "campus" {
  name        = "Campus"
  slug        = "campus"
  description = "All campus SSIDs"
}

resource "netbox_wireless_lan_group" "campus_guest" {
  name      = "Campus guest"
  slug      = "campus-guest"
  parent_id = netbox_wireless_lan_group.campus.id
}
