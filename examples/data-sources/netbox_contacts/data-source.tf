data "netbox_contacts" "acme_noc" {
  filters = [
    { name = "group", value = "acme-corp-noc" },
  ]
}

output "acme_noc_emails" {
  value = data.netbox_contacts.acme_noc.items[*].email
}
