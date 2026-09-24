resource "netbox_owner_group" "infrastructure" {
  name = "Infrastructure"
}

resource "netbox_user_group" "network_ops" {
  name = "network-ops"
}

resource "netbox_user" "jdoe" {
  username = "jdoe"
  password = var.jdoe_password
}

variable "jdoe_password" {
  type      = string
  sensitive = true
}

# An owner (NetBox 4.7+) is a team that can be set as owner_id on most
# objects. Its members are NetBox users and user groups.
resource "netbox_owner" "network_team" {
  name           = "Network team"
  group_id       = netbox_owner_group.infrastructure.id
  description    = "Owns switches, routers and IPAM data"
  user_group_ids = [netbox_user_group.network_ops.id]
  user_ids       = [netbox_user.jdoe.id]
}
