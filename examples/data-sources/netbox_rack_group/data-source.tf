# Look up a rack_group by its unique slug.
data "netbox_rack_group" "example" {
  slug = "compute-racks"
}

# Or by any API filter (see the NetBox filter documentation for /api/dcim/rack-groups/).
data "netbox_rack_group" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "rack_group_id" {
  value = data.netbox_rack_group.example.id
}
