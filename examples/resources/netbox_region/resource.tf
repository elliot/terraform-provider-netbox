resource "netbox_tag" "infra" {
  name = "infra"
  slug = "infra"
}

resource "netbox_region" "europe" {
  name        = "Europe"
  slug        = "europe"
  description = "EMEA operations"
}

# Regions nest: a child region refers to its parent by ID.
resource "netbox_region" "germany" {
  name        = "Germany"
  slug        = "de"
  parent_id   = netbox_region.europe.id
  description = "German data centres"
  tags        = [netbox_tag.infra.slug]
}
