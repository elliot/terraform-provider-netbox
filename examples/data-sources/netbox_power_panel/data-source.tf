data "netbox_site" "fra1" {
  slug = "fra1"
}

# Power panel names are unique per site.
data "netbox_power_panel" "pp_a" {
  filters = [
    { name = "site_id", value = data.netbox_site.fra1.id },
    { name = "name", value = "PP-A" },
  ]
}
