resource "netbox_ipam_role" "production" {
  name        = "Production"
  slug        = "production"
  weight      = 1000
  description = "Production workloads"
}

resource "netbox_ipam_role" "management" {
  name   = "Management"
  slug   = "management"
  weight = 500
}
