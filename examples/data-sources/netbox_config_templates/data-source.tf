data "netbox_config_templates" "attachments" {
  filters = [{ name = "as_attachment", value = "true" }]
}

output "attachment_templates" {
  value = data.netbox_config_templates.attachments.items[*].name
}
