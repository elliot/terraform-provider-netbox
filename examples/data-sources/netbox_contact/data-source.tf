data "netbox_contact" "jane" {
  name = "Jane Doe"
}

resource "netbox_contact_assignment" "example" {
  object_type = "dcim.site"
  object_id   = netbox_site.example.id
  contact_id  = data.netbox_contact.jane.id
}
