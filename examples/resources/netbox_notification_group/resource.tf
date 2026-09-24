resource "netbox_user_group" "netops" {
  name = "netops"
}

resource "netbox_user" "oncall" {
  username = "oncall"
  password = var.oncall_password
}

# Recipients for notification event rules (action_type = "notification").
resource "netbox_notification_group" "netops" {
  name        = "netops"
  description = "Network operations on-call rotation"
  group_ids   = [netbox_user_group.netops.id]
  user_ids    = [netbox_user.oncall.id]
}

resource "netbox_event_rule" "circuit_changes" {
  name               = "circuit-changes"
  object_types       = ["circuits.circuit"]
  event_types        = ["object_created", "object_updated", "object_deleted"]
  action_type        = "notification"
  action_object_type = "extras.notificationgroup"
  action_object_id   = netbox_notification_group.netops.id
}
