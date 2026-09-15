resource "netbox_rack_role" "network" {
  name        = "Network"
  slug        = "network"
  color       = "2196f3"
  description = "Racks dedicated to network equipment"
}

resource "netbox_rack_role" "compute" {
  name        = "Compute"
  slug        = "compute"
  color       = "4caf50"
  description = "Racks dedicated to servers"
}
