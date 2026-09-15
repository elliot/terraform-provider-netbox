data "netbox_prefixes" "all" {}

data "netbox_prefixes" "filtered" {
  filters = [
    { name = "within_include", value = "10.10.0.0/16" },
  ]
  limit = 50
}

output "prefixes_ids" {
  value = data.netbox_prefixes.filtered.items[*].id
}
