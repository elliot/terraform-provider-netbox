resource "netbox_webhook" "slack" {
  name        = "slack-netops"
  payload_url = "https://hooks.slack.com/services/T000/B000/XXXX"
}

# Fire the webhook whenever a device or site is created or updated.
resource "netbox_event_rule" "device_changes" {
  name               = "device-changes"
  description        = "Notify #netops about device and site changes"
  object_types       = ["dcim.device", "dcim.site"]
  event_types        = ["object_created", "object_updated"]
  action_type        = "webhook"
  action_object_type = "extras.webhook"
  action_object_id   = netbox_webhook.slack.id
  enabled            = true
}

# Only trigger when a condition on the object matches.
resource "netbox_event_rule" "active_only" {
  name               = "active-devices-only"
  object_types       = ["dcim.device"]
  event_types        = ["object_updated"]
  action_type        = "webhook"
  action_object_type = "extras.webhook"
  action_object_id   = netbox_webhook.slack.id
  conditions = jsonencode({
    and = [
      { attr = "status.value", value = "active" },
      { attr = "role.slug", value = "core-switch" },
    ]
  })
}
