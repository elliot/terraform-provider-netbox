resource "netbox_tag" "infra" {
  name = "infra"
  slug = "infra"
}

resource "netbox_rack_group" "compute" {
  name        = "Compute racks"
  slug        = "compute-racks"
  description = "Racks hosting compute nodes"
  tags        = [netbox_tag.infra.slug]
}
