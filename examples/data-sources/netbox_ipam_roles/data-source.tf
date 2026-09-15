data "netbox_ipam_roles" "all" {}

data "netbox_ipam_roles" "filtered" {
  filters = [
    { name = "q", value = "prod" },
  ]
  limit = 50
}

output "ipam_roles_ids" {
  value = data.netbox_ipam_roles.filtered.items[*].id
}
