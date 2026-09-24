# All terminations of one L2VPN.
data "netbox_l2vpn_terminations" "tenant_a" {
  filters = [
    { name = "l2vpn_id", value = tostring(netbox_l2vpn.tenant_a.id) },
  ]
}

output "tenant_a_termination_objects" {
  value = data.netbox_l2vpn_terminations.tenant_a.items[*].assigned_object_id
}
