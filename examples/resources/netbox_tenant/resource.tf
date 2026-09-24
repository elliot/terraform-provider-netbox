resource "netbox_tenant_group" "example" {
  name = "Customers"
  slug = "customers"
}

resource "netbox_tenant" "example" {
  name        = "Acme Corp"
  slug        = "acme-corp"
  group_id    = netbox_tenant_group.example.id
  description = "Managed services customer since 2019"
  tags        = ["managed"]
}
