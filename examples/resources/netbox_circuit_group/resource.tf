resource "netbox_tenant" "example" {
  name = "Acme Corp"
  slug = "acme-corp"
}

resource "netbox_circuit_group" "example" {
  name        = "AMS uplinks"
  slug        = "ams-uplinks"
  tenant_id   = netbox_tenant.example.id
  description = "Redundant internet uplinks of the Amsterdam office"
}
