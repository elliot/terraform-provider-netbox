# All terminations of one tunnel.
data "netbox_tunnel_terminations" "gre_branch01" {
  filters = [
    { name = "tunnel_id", value = tostring(netbox_tunnel.gre_branch01.id) },
  ]
}

output "gre_branch01_termination_roles" {
  value = data.netbox_tunnel_terminations.gre_branch01.items[*].role
}
