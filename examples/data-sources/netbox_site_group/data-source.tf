# Look up a site_group by its unique slug.
data "netbox_site_group" "example" {
  slug = "datacenters"
}

# Or by any API filter (see the NetBox filter documentation for /api/dcim/site-groups/).
data "netbox_site_group" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "site_group_id" {
  value = data.netbox_site_group.example.id
}
