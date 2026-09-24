# Look up a single permission by name.
data "netbox_permission" "example" {
  name = "network-ops: manage devices"
}

# Any API filter of /api/users/permissions/ works with filters; the lookup must match exactly one object.
data "netbox_permission" "filtered" {
  filters = [
    { name = "enabled", value = "true" },
  ]
}

output "permission_id" {
  value = data.netbox_permission.example.id
}
