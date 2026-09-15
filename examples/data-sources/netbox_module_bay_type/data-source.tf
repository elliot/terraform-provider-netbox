# Look up a module_bay_type by its unique slug.
data "netbox_module_bay_type" "example" {
  slug = "sfp28-cage"
}

# Or by any API filter (see the NetBox filter documentation for /api/dcim/module-bay-types/).
data "netbox_module_bay_type" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "module_bay_type_id" {
  value = data.netbox_module_bay_type.example.id
}
