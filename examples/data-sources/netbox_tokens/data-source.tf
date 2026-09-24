# All tokens (paginated by NetBox; use limit to cap the result).
data "netbox_tokens" "all" {}

# Filters take any query parameter of /api/users/tokens/.
data "netbox_tokens" "filtered" {
  filters = [
    { name = "user", value = "automation" },
  ]
  limit = 50
}

output "tokens_ids" {
  value = data.netbox_tokens.filtered.items[*].id
}
