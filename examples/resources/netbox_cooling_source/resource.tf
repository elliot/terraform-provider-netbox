resource "netbox_site" "fra1" {
  name = "FRA1"
  slug = "fra1"
}

resource "netbox_location" "plant_room" {
  name    = "Plant room"
  slug    = "fra1-plant-room"
  site_id = netbox_site.fra1.id
}

resource "netbox_tag" "cooling" {
  name = "cooling"
  slug = "cooling"
}

resource "netbox_cooling_source" "chiller_1" {
  site_id          = netbox_site.fra1.id
  location_id      = netbox_location.plant_room.id
  name             = "CH-01"
  type             = "chiller"
  status           = "active"
  fluid_type       = "water-glycol"
  cooling_capacity = 250
  description      = "Primary chiller"
  tags             = [netbox_tag.cooling.slug]
}
