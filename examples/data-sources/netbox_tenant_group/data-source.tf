data "netbox_tenant_group" "customers" {
  slug = "customers"
}

resource "netbox_tenant" "example" {
  name     = "Acme Corp"
  slug     = "acme-corp"
  group_id = data.netbox_tenant_group.customers.id
}
