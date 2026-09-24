data "netbox_contact_assignment" "example" {
  id = 44
}

output "assignment_contact_id" {
  value = data.netbox_contact_assignment.example.contact_id
}
