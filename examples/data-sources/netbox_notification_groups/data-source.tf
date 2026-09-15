data "netbox_notification_groups" "all" {}

output "notification_group_names" {
  value = data.netbox_notification_groups.all.items[*].name
}
