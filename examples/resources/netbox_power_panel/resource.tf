resource "netbox_site" "fra1" {
  name = "FRA1"
  slug = "fra1"
}

resource "netbox_location" "electrical_room" {
  name    = "Electrical room"
  slug    = "fra1-electrical-room"
  site_id = netbox_site.fra1.id
}

resource "netbox_tag" "power" {
  name = "power"
  slug = "power"
}

resource "netbox_power_panel" "pdu_a" {
  site_id     = netbox_site.fra1.id
  location_id = netbox_location.electrical_room.id
  name        = "PP-A"
  description = "Distribution panel, A side"
  tags        = [netbox_tag.power.slug]
}
