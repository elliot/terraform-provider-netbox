data "netbox_ip_ranges" "all" {}

data "netbox_ip_ranges" "filtered" {
  filters = [
    { name = "status", value = "active" },
  ]
  limit = 50
}

output "ip_ranges_ids" {
  value = data.netbox_ip_ranges.filtered.items[*].id
}
