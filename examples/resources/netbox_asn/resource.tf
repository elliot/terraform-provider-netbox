resource "netbox_rir" "rfc6996" {
  name       = "RFC 6996"
  slug       = "rfc-6996"
  is_private = true
}

resource "netbox_site" "dc1" {
  name = "DC1"
  slug = "dc1"
}

# ASNs are unique across NetBox; the RIR is optional but recommended.
resource "netbox_asn" "dc1" {
  asn         = 4200000001
  rir_id      = netbox_rir.rfc6996.id
  site_ids    = [netbox_site.dc1.id]
  description = "DC1 fabric underlay"
}
