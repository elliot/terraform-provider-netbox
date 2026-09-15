# Tokens have no unique name; look one up by ID or by API filters.
data "netbox_token" "example" {
  id = 123
}

# Any API filter of /api/users/tokens/ works with filters; the lookup must match exactly one object.
data "netbox_token" "filtered" {
  filters = [
    { name = "user", value = "automation" },
  ]
}

output "token_id" {
  value = data.netbox_token.example.id
}
