# Look up a site by its unique slug.
data "netbox_site" "example" {
  slug = "fra1"
}

# Or by any API filter (see the NetBox filter documentation for /api/dcim/sites/).
data "netbox_site" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "site_id" {
  value = data.netbox_site.example.id
}
