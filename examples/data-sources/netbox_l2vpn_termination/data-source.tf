# Terminations have no natural key: look them up by id or by filters.
data "netbox_l2vpn_termination" "by_id" {
  id = 123
}

data "netbox_l2vpn_termination" "vlan" {
  filters = [
    { name = "l2vpn", value = "tenant-a-overlay" },
    { name = "assigned_object_type", value = "ipam.vlan" },
  ]
}
