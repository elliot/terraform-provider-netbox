# All owner groups (paginated by NetBox; use limit to cap the result).
data "netbox_owner_groups" "all" {}

# Filters take any query parameter of /api/users/owner-groups/.
data "netbox_owner_groups" "filtered" {
  filters = [
    { name = "q", value = "infra" },
  ]
  limit = 50
}

output "owner_groups_ids" {
  value = data.netbox_owner_groups.filtered.items[*].id
}
