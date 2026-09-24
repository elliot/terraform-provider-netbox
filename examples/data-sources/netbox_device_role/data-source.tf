# Look up a device_role by its unique slug.
data "netbox_device_role" "example" {
  slug = "access-switch"
}

# Or by any API filter (see the NetBox filter documentation for /api/dcim/device-roles/).
data "netbox_device_role" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "device_role_id" {
  value = data.netbox_device_role.example.id
}
