# All owners (paginated by NetBox; use limit to cap the result).
data "netbox_owners" "all" {}

# Filters take any query parameter of /api/users/owners/.
data "netbox_owners" "filtered" {
  filters = [
    { name = "group", value = "Infrastructure" },
  ]
  limit = 50
}

output "owners_ids" {
  value = data.netbox_owners.filtered.items[*].id
}
