# All users (paginated by NetBox; use limit to cap the result).
data "netbox_users" "all" {}

# Filters take any query parameter of /api/users/users/.
data "netbox_users" "filtered" {
  filters = [
    { name = "group", value = "network-ops" },
  ]
  limit = 50
}

output "users_ids" {
  value = data.netbox_users.filtered.items[*].id
}
