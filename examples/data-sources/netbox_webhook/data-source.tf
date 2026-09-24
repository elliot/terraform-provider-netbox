data "netbox_webhook" "slack" {
  name = "slack-netops"
}

# Attach an event rule to a webhook managed outside Terraform.
resource "netbox_event_rule" "vm_changes" {
  name               = "vm-changes"
  object_types       = ["virtualization.virtualmachine"]
  event_types        = ["object_created", "object_deleted"]
  action_type        = "webhook"
  action_object_type = "extras.webhook"
  action_object_id   = data.netbox_webhook.slack.id
}
