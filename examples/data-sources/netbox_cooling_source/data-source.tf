data "netbox_site" "fra1" {
  slug = "fra1"
}

# Cooling source names are unique per site.
data "netbox_cooling_source" "chiller_1" {
  filters = [
    { name = "site_id", value = data.netbox_site.fra1.id },
    { name = "name", value = "CH-01" },
  ]
}
