resource "netbox_contact_group" "acme" {
  name        = "Acme Corp"
  slug        = "acme-corp"
  description = "Contacts at Acme Corp"
}

resource "netbox_contact_group" "acme_noc" {
  name      = "Acme Corp NOC"
  slug      = "acme-corp-noc"
  parent_id = netbox_contact_group.acme.id
}
