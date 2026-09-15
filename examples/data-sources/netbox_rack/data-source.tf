# Rack names are unique per location, so narrow the lookup with a filter.
data "netbox_site" "fra1" {
  slug = "fra1"
}

data "netbox_rack" "a01" {
  filters = [
    { name = "site_id", value = data.netbox_site.fra1.id },
    { name = "name", value = "A01" },
  ]
}

output "rack_u_height" {
  value = data.netbox_rack.a01.u_height
}
