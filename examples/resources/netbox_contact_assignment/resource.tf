resource "netbox_site" "example" {
  name = "Amsterdam office"
  slug = "ams-office"
}

resource "netbox_contact" "example" {
  name  = "Jane Doe"
  email = "jane.doe@acme.example"
}

resource "netbox_contact_role" "example" {
  name = "Technical"
  slug = "technical"
}

# Contacts can be assigned to any object type that supports contacts
# (dcim.site, dcim.device, circuits.circuit, tenancy.tenant, ...).
resource "netbox_contact_assignment" "example" {
  object_type = "dcim.site"
  object_id   = netbox_site.example.id
  contact_id  = netbox_contact.example.id
  role_id     = netbox_contact_role.example.id
  priority    = "primary"
}
