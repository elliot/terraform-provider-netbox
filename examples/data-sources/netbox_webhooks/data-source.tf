# Free-text search across name and payload URL.
data "netbox_webhooks" "example_com" {
  filters = [{ name = "q", value = "example.com" }]
}

output "example_webhook_urls" {
  value = data.netbox_webhooks.example_com.items[*].payload_url
}
