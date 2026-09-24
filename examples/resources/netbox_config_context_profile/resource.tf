# A profile validates the data of the config contexts assigned to it.
resource "netbox_config_context_profile" "ntp" {
  name        = "ntp"
  description = "Schema for NTP settings"
  schema = jsonencode({
    type = "object"
    properties = {
      ntp_servers = { type = "array", items = { type = "string" } }
    }
    required = ["ntp_servers"]
  })
}

resource "netbox_config_context" "ntp_emea" {
  name       = "ntp-emea"
  profile_id = netbox_config_context_profile.ntp.id
  data       = jsonencode({ ntp_servers = ["10.10.0.1", "10.10.0.2"] })
}
