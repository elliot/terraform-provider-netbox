resource "netbox_tenant_group" "customers" {
  name        = "Customers"
  slug        = "customers"
  description = "External customers"
}

resource "netbox_tenant_group" "enterprise" {
  name      = "Enterprise customers"
  slug      = "enterprise-customers"
  parent_id = netbox_tenant_group.customers.id
}
