resource "netbox_site" "fra1" {
  name = "FRA1"
  slug = "fra1"
}

resource "netbox_power_panel" "pdu_a" {
  site_id = netbox_site.fra1.id
  name    = "PP-A"
}

resource "netbox_rack" "a01" {
  name    = "A01"
  site_id = netbox_site.fra1.id
}

resource "netbox_tag" "power" {
  name = "power"
  slug = "power"
}

# Feeds from the panel to the rack. The A/B pair share a rack but come from
# different panels in a real deployment.
resource "netbox_power_feed" "a01_a" {
  power_panel_id  = netbox_power_panel.pdu_a.id
  rack_id         = netbox_rack.a01.id
  name            = "A01-A"
  status          = "active"
  type            = "primary"
  supply          = "ac"
  phase           = "single-phase"
  voltage         = 230
  amperage        = 32
  max_utilization = 80
  description     = "Primary feed for rack A01"
  tags            = [netbox_tag.power.slug]
}

resource "netbox_power_feed" "a01_b" {
  power_panel_id = netbox_power_panel.pdu_a.id
  rack_id        = netbox_rack.a01.id
  name           = "A01-B"
  type           = "redundant"
  supply         = "ac"
  phase          = "single-phase"
  voltage        = 230
  amperage       = 32
}
