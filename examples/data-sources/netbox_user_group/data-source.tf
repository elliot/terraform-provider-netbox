# Look up a single user group by name.
data "netbox_user_group" "example" {
  name = "network-ops"
}

# Any API filter of /api/users/groups/ works with filters; the lookup must match exactly one object.
data "netbox_user_group" "filtered" {
  filters = [
    { name = "q", value = "ops" },
  ]
}

output "user_group_id" {
  value = data.netbox_user_group.example.id
}
