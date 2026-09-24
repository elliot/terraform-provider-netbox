resource "netbox_rir" "rfc6996" {
  name       = "RFC 6996"
  slug       = "rfc-6996"
  is_private = true
}

resource "netbox_asn_range" "fabric" {
  name        = "Fabric underlay ASNs"
  slug        = "fabric-underlay"
  rir_id      = netbox_rir.rfc6996.id
  start       = 4200000000
  end         = 4200000999
  description = "One private ASN per leaf switch"
}
