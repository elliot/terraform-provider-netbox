resource "netbox_webhook" "slack" {
  name             = "slack-netops"
  payload_url      = "https://hooks.slack.com/services/T000/B000/XXXX"
  http_method      = "POST"
  http_content_type = "application/json"
  body_template    = <<-EOT
    {"text": "{{ event }} {{ model }} {{ data.name }} by {{ username }}"}
  EOT
  ssl_verification = true
  timeout          = 10
}

# A signed webhook with custom headers; the secret is used for X-Hook-Signature.
resource "netbox_webhook" "cmdb" {
  name               = "cmdb-sync"
  description        = "Push changes to the CMDB"
  payload_url        = "https://cmdb.example.com/api/netbox"
  http_method        = "PUT"
  additional_headers = "Authorization: Bearer ${var.cmdb_token}\nX-Source: netbox"
  secret             = var.webhook_secret
}
