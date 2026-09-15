resource "netbox_tenant" "acme" {
  name = "ACME Corp"
  slug = "acme"
}

resource "netbox_route_target" "acme_import" {
  name        = "65000:100"
  tenant_id   = netbox_tenant.acme.id
  description = "ACME L3VPN import"
}

resource "netbox_route_target" "acme_export" {
  name        = "65000:101"
  tenant_id   = netbox_tenant.acme.id
  description = "ACME L3VPN export"
}
