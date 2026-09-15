resource "netbox_site_group" "corporate" {
  name        = "Corporate"
  slug        = "corporate"
  description = "Corporate offices and campuses"
}

resource "netbox_site_group" "branch_offices" {
  name        = "Branch offices"
  slug        = "branch-offices"
  parent_id   = netbox_site_group.corporate.id
  description = "Small branch offices"
  comments    = "Managed by Terraform"
}
