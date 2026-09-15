# Event rules that fire on object deletion.
data "netbox_event_rules" "on_delete" {
  filters = [{ name = "event_type", value = "object_deleted" }]
}

output "delete_rules" {
  value = data.netbox_event_rules.on_delete.items[*].name
}
