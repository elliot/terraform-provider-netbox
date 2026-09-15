# All permissions (paginated by NetBox; use limit to cap the result).
data "netbox_permissions" "all" {}

# Filters take any query parameter of /api/users/permissions/.
data "netbox_permissions" "filtered" {
  filters = [
    { name = "enabled", value = "true" },
  ]
  limit = 50
}

output "permissions_ids" {
  value = data.netbox_permissions.filtered.items[*].id
}
