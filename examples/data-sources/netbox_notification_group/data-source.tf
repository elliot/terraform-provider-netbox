data "netbox_notification_group" "netops" {
  filters = [{ name = "name", value = "netops" }]
}

output "netops_user_ids" {
  value = data.netbox_notification_group.netops.user_ids
}
