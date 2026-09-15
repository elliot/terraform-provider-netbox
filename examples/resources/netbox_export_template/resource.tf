# Export sites as CSV. `queryset` holds the objects selected in the UI.
resource "netbox_export_template" "sites_csv" {
  name           = "sites-csv"
  description    = "Site inventory for the facilities team"
  object_types   = ["dcim.site"]
  mime_type      = "text/csv"
  file_name      = "sites"
  file_extension = "csv"
  as_attachment  = true
  template_code  = <<-EOT
    name,status,region,facility
    {% for site in queryset %}{{ site.name }},{{ site.status }},{{ site.region }},{{ site.facility }}
    {% endfor %}
  EOT
}
