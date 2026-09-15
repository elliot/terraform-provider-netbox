data "netbox_webhooks" "example_com" {
  filters = [{ name = "payload_url__ic", value = "example.com" }]
}

output "example_webhook_urls" {
  value = data.netbox_webhooks.example_com.items[*].payload_url
}
