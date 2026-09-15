# All custom fields defined on sites.
data "netbox_custom_fields" "site" {
  filters = [{ name = "object_type", value = "dcim.site" }]
}

output "site_custom_field_names" {
  value = data.netbox_custom_fields.site.items[*].name
}
