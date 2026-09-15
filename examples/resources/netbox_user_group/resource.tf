resource "netbox_user_group" "network_ops" {
  name        = "network-ops"
  description = "Network operations staff"
}

# Permissions are assigned from netbox_permission (group_ids), not from the group.
resource "netbox_permission" "network_ops_read" {
  name         = "network-ops: read-only"
  object_types = ["dcim.device", "dcim.interface", "ipam.prefix"]
  actions      = ["view"]
  group_ids    = [netbox_user_group.network_ops.id]
}
