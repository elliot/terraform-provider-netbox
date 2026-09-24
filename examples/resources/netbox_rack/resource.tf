resource "netbox_site" "fra1" {
  name = "FRA1"
  slug = "fra1"
}

resource "netbox_location" "cage_a" {
  name    = "Cage A"
  slug    = "fra1-cage-a"
  site_id = netbox_site.fra1.id
}

resource "netbox_tenant" "acme" {
  name = "ACME Corp"
  slug = "acme"
}

resource "netbox_rack_role" "compute" {
  name  = "Compute"
  slug  = "compute"
  color = "4caf50"
}

resource "netbox_tag" "production" {
  name = "production"
  slug = "production"
}

resource "netbox_rack" "a01" {
  name           = "A01"
  site_id        = netbox_site.fra1.id
  location_id    = netbox_location.cage_a.id
  tenant_id      = netbox_tenant.acme.id
  role_id        = netbox_rack_role.compute.id
  status         = "active"
  facility_id    = "FRA1-2A-01"
  serial         = "RK-2024-00017"
  asset_tag      = "A-10017"
  form_factor    = "4-post-cabinet"
  width          = 19
  u_height       = 42
  starting_unit  = 1
  outer_width    = 600
  outer_depth    = 1070
  outer_unit     = "mm"
  mounting_depth = 900
  max_weight     = 1300
  weight_unit    = "kg"
  airflow        = "front-to-rear"
  description    = "First compute rack in cage A"
  comments       = "Managed by Terraform"
  tags           = [netbox_tag.production.slug]
}
