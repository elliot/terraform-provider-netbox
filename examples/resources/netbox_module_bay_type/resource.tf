resource "netbox_manufacturer" "example" {
  name = "Example Networks"
  slug = "example-networks"
}

# Module bay types (NetBox 4.5+) classify module bays, e.g. by connector.
resource "netbox_module_bay_type" "sfp28" {
  name            = "SFP28 cage"
  slug            = "sfp28-cage"
  manufacturer_id = netbox_manufacturer.example.id
  color           = "9c27b0"
  description     = "Single SFP28 cage"
}
