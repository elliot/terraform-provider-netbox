resource "netbox_tag" "vendor" {
  name = "vendor"
  slug = "vendor"
}

resource "netbox_manufacturer" "juniper" {
  name        = "Juniper Networks"
  slug        = "juniper"
  description = "Routers, switches and firewalls"
  tags        = [netbox_tag.vendor.slug]
}
