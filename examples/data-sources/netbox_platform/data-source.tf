# Look up a platform by its unique slug.
data "netbox_platform" "example" {
  slug = "junos"
}

# Or by any API filter (see the NetBox filter documentation for /api/dcim/platforms/).
data "netbox_platform" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "platform_id" {
  value = data.netbox_platform.example.id
}
