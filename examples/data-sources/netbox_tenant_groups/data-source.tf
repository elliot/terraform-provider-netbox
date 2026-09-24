data "netbox_tenant_groups" "all" {}

output "tenant_groups" {
  value = { for g in data.netbox_tenant_groups.all.items : g.slug => g.tenant_count }
}
