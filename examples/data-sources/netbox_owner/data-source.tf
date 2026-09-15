# Look up a single owner by name.
data "netbox_owner" "example" {
  name = "Network team"
}

# Any API filter of /api/users/owners/ works with filters; the lookup must match exactly one object.
data "netbox_owner" "filtered" {
  filters = [
    { name = "group", value = "Infrastructure" },
  ]
}

output "owner_id" {
  value = data.netbox_owner.example.id
}
