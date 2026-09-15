# Look up a rack_type by its unique slug.
data "netbox_rack_type" "example" {
  slug = "apc-netshelter-sx-42u"
}

# Or by any API filter (see the NetBox filter documentation for /api/dcim/rack-types/).
data "netbox_rack_type" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "rack_type_id" {
  value = data.netbox_rack_type.example.id
}
