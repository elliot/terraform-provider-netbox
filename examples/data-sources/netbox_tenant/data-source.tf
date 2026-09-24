data "netbox_tenant" "acme" {
  slug = "acme-corp"
}

resource "netbox_circuit" "example" {
  cid         = "LUM-DIA-48213"
  provider_id = netbox_provider.example.id
  type_id     = netbox_circuit_type.example.id
  tenant_id   = data.netbox_tenant.acme.id
}
