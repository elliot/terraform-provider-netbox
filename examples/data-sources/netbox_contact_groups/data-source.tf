data "netbox_contact_groups" "acme" {
  filters = [
    { name = "parent", value = "acme-corp" },
  ]
}

output "acme_contact_groups" {
  value = data.netbox_contact_groups.acme.items[*].slug
}
