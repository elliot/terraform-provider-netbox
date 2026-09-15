data "netbox_export_template" "sites_csv" {
  name = "sites-csv"
}

output "sites_csv_mime_type" {
  value = data.netbox_export_template.sites_csv.mime_type
}
