data "netbox_asns" "all" {}

data "netbox_asns" "filtered" {
  filters = [
    { name = "asn__gte", value = "4200000000" },
  ]
  limit = 50
}

output "asns_ids" {
  value = data.netbox_asns.filtered.items[*].id
}
