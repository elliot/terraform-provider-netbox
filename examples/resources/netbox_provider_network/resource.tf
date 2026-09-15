resource "netbox_provider" "example" {
  name = "Lumen"
  slug = "lumen"
}

resource "netbox_provider_network" "example" {
  provider_id = netbox_provider.example.id
  name        = "Lumen MPLS EU"
  service_id  = "MPLS-EU-01"
  description = "Provider MPLS cloud used as the Z side of branch circuits"
}
