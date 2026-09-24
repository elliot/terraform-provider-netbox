data "netbox_asn_ranges" "all" {}

data "netbox_asn_ranges" "filtered" {
  filters = [
    { name = "rir", value = "rfc-6996" },
  ]
  limit = 50
}

output "asn_ranges_ids" {
  value = data.netbox_asn_ranges.filtered.items[*].id
}
