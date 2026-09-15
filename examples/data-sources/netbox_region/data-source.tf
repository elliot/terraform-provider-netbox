# Look up a region by its unique slug.
data "netbox_region" "example" {
  slug = "europe"
}

# Or by any API filter (see the NetBox filter documentation for /api/dcim/regions/).
data "netbox_region" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "region_id" {
  value = data.netbox_region.example.id
}
