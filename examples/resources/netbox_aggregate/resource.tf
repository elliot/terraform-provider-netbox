resource "netbox_rir" "rfc1918" {
  name       = "RFC 1918"
  slug       = "rfc-1918"
  is_private = true
}

resource "netbox_tenant" "acme" {
  name = "ACME Corp"
  slug = "acme"
}

# Aggregates may not overlap each other; they are the root of the prefix tree.
resource "netbox_aggregate" "ten" {
  prefix      = "10.0.0.0/8"
  rir_id      = netbox_rir.rfc1918.id
  tenant_id   = netbox_tenant.acme.id
  date_added  = "2024-01-15"
  description = "Corporate private space"
}
