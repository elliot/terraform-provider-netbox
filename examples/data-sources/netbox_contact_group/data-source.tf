data "netbox_contact_group" "acme_noc" {
  slug = "acme-corp-noc"
}

resource "netbox_contact" "example" {
  name      = "Jane Doe"
  group_ids = [data.netbox_contact_group.acme_noc.id]
}
