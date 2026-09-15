# Look up a single user by username.
data "netbox_user" "example" {
  username = "jdoe"
}

# Any API filter of /api/users/users/ works with filters; the lookup must match exactly one object.
data "netbox_user" "filtered" {
  filters = [
    { name = "group", value = "network-ops" },
  ]
}

output "user_id" {
  value = data.netbox_user.example.id
}
