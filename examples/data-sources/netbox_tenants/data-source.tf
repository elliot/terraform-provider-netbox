data "netbox_tenants" "customers" {
  filters = [
    { name = "group", value = "customers" },
  ]
  limit = 200
}

output "customer_slugs" {
  value = data.netbox_tenants.customers.items[*].slug
}
