variable "jdoe_password" {
  type      = string
  sensitive = true
}

resource "netbox_user_group" "network_ops" {
  name = "network-ops"
}

# The password is write-only: NetBox never returns it, so it is kept from
# the configuration and re-sent on every update. Group membership is managed
# here; object permissions are assigned from netbox_permission.
resource "netbox_user" "jdoe" {
  username   = "jdoe"
  password   = var.jdoe_password
  first_name = "Jane"
  last_name  = "Doe"
  email      = "jane.doe@example.com"
  is_active  = true
  group_ids  = [netbox_user_group.network_ops.id]
}
