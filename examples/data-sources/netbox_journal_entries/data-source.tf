# Journal entries of one site, newest first.
data "netbox_journal_entries" "site" {
  filters = [
    { name = "assigned_object_type", value = "dcim.site" },
    { name = "assigned_object_id", value = tostring(netbox_site.example.id) },
    { name = "ordering", value = "-created" },
  ]
  limit = 20
}

output "site_journal" {
  value = data.netbox_journal_entries.site.items[*].comments
}
