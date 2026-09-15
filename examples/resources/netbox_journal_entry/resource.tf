resource "netbox_site" "example" {
  name = "Amsterdam"
  slug = "ams"
}

# Journal entries attach free-form notes to any object. created_by defaults
# to the API token's user.
resource "netbox_journal_entry" "maintenance" {
  assigned_object_type = "dcim.site"
  assigned_object_id   = netbox_site.example.id
  kind                 = "warning"
  comments             = "Scheduled power maintenance 2026-10-01 02:00-04:00 CEST"
}
