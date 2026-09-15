data "netbox_virtual_circuits" "on_mpls" {
  filters = [
    { name = "provider_network_id", value = tostring(netbox_provider_network.example.id) },
  ]
}

output "mpls_virtual_circuits" {
  value = data.netbox_virtual_circuits.on_mpls.items[*].cid
}
