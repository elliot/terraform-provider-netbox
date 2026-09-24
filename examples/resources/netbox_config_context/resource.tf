resource "netbox_region" "emea" {
  name = "EMEA"
  slug = "emea"
}

resource "netbox_device_role" "core" {
  name = "Core switch"
  slug = "core-switch"
}

# Rendered into the config context of every device in EMEA; higher weight wins.
resource "netbox_config_context" "ntp_emea" {
  name        = "ntp-emea"
  description = "Regional NTP and syslog servers"
  weight      = 1000
  is_active   = true
  region_ids  = [netbox_region.emea.id]
  role_ids    = [netbox_device_role.core.id]
  data = jsonencode({
    ntp_servers = ["10.10.0.1", "10.10.0.2"]
    syslog      = { host = "10.10.0.3", port = 514 }
  })
}
