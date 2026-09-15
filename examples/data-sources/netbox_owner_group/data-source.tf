# Look up a single owner group by name.
data "netbox_owner_group" "example" {
  name = "Infrastructure"
}

# Any API filter of /api/users/owner-groups/ works with filters; the lookup must match exactly one object.
data "netbox_owner_group" "filtered" {
  filters = [
    { name = "q", value = "infra" },
  ]
}

output "owner_group_id" {
  value = data.netbox_owner_group.example.id
}
