resource "netbox_site" "fra1" {
  name = "FRA1"
  slug = "fra1"
}

resource "netbox_cooling_source" "chiller_1" {
  site_id = netbox_site.fra1.id
  name    = "CH-01"
  type    = "chiller"
}

resource "netbox_rack" "a01" {
  name    = "A01"
  site_id = netbox_site.fra1.id
}

resource "netbox_cooling_feed" "a01_loop" {
  cooling_source_id = netbox_cooling_source.chiller_1.id
  rack_id           = netbox_rack.a01.id
  name              = "CH-01/A01"
  status            = "active"
  cooling_capacity  = 30
  max_flow          = 12.5
  max_flow_unit     = "lpm"
  description       = "Coolant loop to rack A01"
}
