# Look up a manufacturer by its unique slug.
data "netbox_manufacturer" "example" {
  slug = "juniper"
}

# Or by any API filter (see the NetBox filter documentation for /api/dcim/manufacturers/).
data "netbox_manufacturer" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "manufacturer_id" {
  value = data.netbox_manufacturer.example.id
}
