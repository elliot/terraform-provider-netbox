data "netbox_export_templates" "site" {
  filters = [{ name = "object_type", value = "dcim.site" }]
}

output "site_export_templates" {
  value = data.netbox_export_templates.site.items[*].name
}
