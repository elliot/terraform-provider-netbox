data "netbox_contact_roles" "all" {}

output "contact_roles" {
  value = data.netbox_contact_roles.all.items[*].slug
}
