resource "netbox_user_group" "network_ops" {
  name = "network-ops"
}

resource "netbox_user" "automation" {
  username = "automation"
  password = var.automation_password
}

variable "automation_password" {
  type      = string
  sensitive = true
}

# object_types use the app_label.model form; actions are view, add, change,
# delete plus any custom action string. constraints is a Django queryset
# filter (an object or a list of objects) limiting the matching rows.
resource "netbox_permission" "network_ops_write" {
  name         = "network-ops: manage devices"
  description  = "Full control over devices and interfaces"
  object_types = ["dcim.device", "dcim.interface"]
  actions      = ["view", "add", "change", "delete"]
  group_ids    = [netbox_user_group.network_ops.id]
}

resource "netbox_permission" "automation_active_sites" {
  name         = "automation: view active sites"
  object_types = ["dcim.site"]
  actions      = ["view"]
  constraints  = jsonencode({ status = "active" })
  user_ids     = [netbox_user.automation.id]
}
