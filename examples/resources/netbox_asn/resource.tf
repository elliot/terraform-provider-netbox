resource "netbox_rir" "rfc6996" {
  name       = "RFC 6996"
  slug       = "rfc-6996"
  is_private = true
}

# ASNs are unique across NetBox; the RIR is optional but recommended.
resource "netbox_asn" "dc1" {
  asn         = 4200000001
  rir_id      = netbox_rir.rfc6996.id
  description = "DC1 fabric underlay"
}

# Sites reference their ASNs through netbox_site.asn_ids; netbox_asn exposes
# the same relation read-only as site_ids.
resource "netbox_site" "dc1" {
  name    = "DC1"
  slug    = "dc1"
  asn_ids = [netbox_asn.dc1.id]
}
