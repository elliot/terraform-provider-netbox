data "netbox_event_rule" "device_changes" {
  name = "device-changes"
}

output "device_changes_enabled" {
  value = data.netbox_event_rule.device_changes.enabled
}
