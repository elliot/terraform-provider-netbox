resource "netbox_contact_group" "example" {
  name = "Acme Corp"
  slug = "acme-corp"
}

resource "netbox_contact" "example" {
  name      = "Jane Doe"
  group_ids = [netbox_contact_group.example.id]
  title     = "Network Operations Manager"
  phone     = "+31 20 555 0100"
  email     = "jane.doe@acme.example"
  address   = "Herengracht 1, 1015 BA Amsterdam"
  link      = "https://acme.example/people/jane-doe"
}
