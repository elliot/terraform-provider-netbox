data "netbox_l2vpns" "all" {}

data "netbox_l2vpns" "filtered" {
  filters = [
    { name = "type", value = "vxlan-evpn" },
  ]
  limit = 50
}

output "l2vpns_names" {
  value = data.netbox_l2vpns.filtered.items[*].name
}
