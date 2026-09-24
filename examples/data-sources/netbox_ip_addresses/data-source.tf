data "netbox_ip_addresses" "all" {}

data "netbox_ip_addresses" "filtered" {
  filters = [
    { name = "parent", value = "10.10.20.0/24" },
  ]
  limit = 50
}

output "ip_addresses_ids" {
  value = data.netbox_ip_addresses.filtered.items[*].id
}
