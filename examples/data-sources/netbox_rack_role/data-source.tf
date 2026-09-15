# Look up a rack_role by its unique slug.
data "netbox_rack_role" "example" {
  slug = "compute"
}

# Or by any API filter (see the NetBox filter documentation for /api/dcim/rack-roles/).
data "netbox_rack_role" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "rack_role_id" {
  value = data.netbox_rack_role.example.id
}
