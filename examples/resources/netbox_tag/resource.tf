resource "netbox_tag" "managed" {
  name        = "Managed by Terraform"
  slug        = "managed-by-terraform"
  color       = "2196f3"
  description = "Objects whose lifecycle is owned by Terraform"
}

# Restrict a tag to certain object types.
resource "netbox_tag" "pci" {
  name         = "PCI"
  slug         = "pci"
  color        = "f44336"
  object_types = ["dcim.device", "virtualization.virtualmachine"]
  weight       = 1000
}

resource "netbox_site" "example" {
  name = "Amsterdam"
  slug = "ams"
  tags = [netbox_tag.managed.slug]
}
