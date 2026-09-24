# Technical contacts of a site.
data "netbox_contact_assignments" "ams_technical" {
  filters = [
    { name = "object_type", value = "dcim.site" },
    { name = "object_id", value = tostring(netbox_site.example.id) },
    { name = "role", value = "technical" },
  ]
}

output "ams_technical_contact_ids" {
  value = data.netbox_contact_assignments.ams_technical.items[*].contact_id
}
