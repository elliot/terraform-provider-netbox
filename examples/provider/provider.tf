terraform {
  required_providers {
    netbox = {
      source  = "elliot/netbox"
      version = "~> 0.1"
    }
  }
}

# Credentials can also come from NETBOX_SERVER_URL and NETBOX_API_TOKEN.
provider "netbox" {
  server_url = "https://netbox.example.com"
  api_token  = var.netbox_token # a v2 token: nbt_<key>.<secret>

  # Client-side pacing shared by every resource (all optional).
  requests_per_second = 10    # token bucket, 0 disables
  request_burst       = 10
  request_delay_ms    = 0     # fixed artificial delay before every request
  write_delay_ms      = 0     # extra delay before POST/PUT/PATCH/DELETE
  serialize_requests  = false # force one request at a time
  max_retries         = 4     # 429/5xx are retried with exponential backoff
  retry_wait_min_ms   = 1000
  retry_wait_max_ms   = 30000
}

variable "netbox_token" {
  type      = string
  sensitive = true
}
