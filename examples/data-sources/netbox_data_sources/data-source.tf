data "netbox_data_sources" "all" {}

data "netbox_data_sources" "filtered" {
  filters = [
    { name = "type", value = "git" },
  ]
  limit = 50
}

output "data_sources_ids" {
  value = data.netbox_data_sources.filtered.items[*].id
}
