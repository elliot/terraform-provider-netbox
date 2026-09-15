# Terminations have no natural key: look them up by id or by filters.
data "netbox_tunnel_termination" "by_id" {
  id = 123
}

data "netbox_tunnel_termination" "hub" {
  filters = [
    { name = "tunnel", value = "gre-branch01" },
    { name = "role", value = "hub" },
  ]
}

output "hub_interface_id" {
  value = data.netbox_tunnel_termination.hub.termination_id
}
