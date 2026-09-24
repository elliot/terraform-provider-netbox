resource "netbox_tag" "structured" {
  name = "structured-cabling"
  slug = "structured-cabling"
}

# Cable bundles (NetBox 4.5+) group cables that run together, e.g. a trunk.
resource "netbox_cable_bundle" "a01_trunk" {
  name        = "A01 trunk"
  description = "24-pair trunk from rack A01 to the MDA"
  comments    = "Managed by Terraform"
  tags        = [netbox_tag.structured.slug]
}
