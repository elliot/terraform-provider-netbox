resource "netbox_region" "germany" {
  name = "Germany"
  slug = "de"
}

resource "netbox_site_group" "datacenters" {
  name = "Data centres"
  slug = "datacenters"
}

resource "netbox_tenant" "acme" {
  name = "ACME Corp"
  slug = "acme"
}

resource "netbox_tag" "production" {
  name = "production"
  slug = "production"
}

resource "netbox_site" "fra1" {
  name             = "FRA1"
  slug             = "fra1"
  status           = "active"
  region_id        = netbox_region.germany.id
  group_id         = netbox_site_group.datacenters.id
  tenant_id        = netbox_tenant.acme.id
  facility         = "Interxion FRA1"
  time_zone        = "Europe/Berlin"
  description      = "Primary data centre in Frankfurt"
  physical_address = "Hanauer Landstraße 300, 60314 Frankfurt am Main, Germany"
  shipping_address = "Hanauer Landstraße 300, 60314 Frankfurt am Main, Germany (loading dock B)"
  latitude         = 50.114500
  longitude        = 8.735500
  comments         = "Managed by Terraform"
  tags             = [netbox_tag.production.slug]
}
