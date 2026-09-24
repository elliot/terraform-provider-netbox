# Hand-maintained example for netbox_available_asn.

resource "netbox_rir" "private" {
  name       = "RFC 6996"
  slug       = "rfc6996"
  is_private = true
}

resource "netbox_asn_range" "private_32bit" {
  name   = "Private 32-bit ASNs"
  slug   = "private-32bit"
  rir_id = netbox_rir.private.id
  start  = 4200000000
  end    = 4200000999
}

resource "netbox_tenant" "acme" {
  name = "ACME"
  slug = "acme"
}

# Take the next free AS number from the range.
resource "netbox_available_asn" "acme_edge" {
  asn_range_id = netbox_asn_range.private_32bit.id
  tenant_id    = netbox_tenant.acme.id
  description  = "ACME edge routers"
}

output "acme_edge_asn" {
  value = netbox_available_asn.acme_edge.asn # e.g. 4200000000
}
