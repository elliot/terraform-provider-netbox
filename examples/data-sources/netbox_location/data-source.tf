# Look up a location by its unique slug.
data "netbox_location" "example" {
  slug = "fra1-cage-a"
}

# Or by any API filter (see the NetBox filter documentation for /api/dcim/locations/).
data "netbox_location" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "location_id" {
  value = data.netbox_location.example.id
}
