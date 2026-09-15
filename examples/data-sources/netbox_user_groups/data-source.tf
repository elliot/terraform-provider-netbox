# All user groups (paginated by NetBox; use limit to cap the result).
data "netbox_user_groups" "all" {}

# Filters take any query parameter of /api/users/groups/.
data "netbox_user_groups" "filtered" {
  filters = [
    { name = "q", value = "ops" },
  ]
  limit = 50
}

output "user_groups_ids" {
  value = data.netbox_user_groups.filtered.items[*].id
}
