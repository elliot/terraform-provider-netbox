data "netbox_config_context_profile" "ntp" {
  name = "ntp"
}

resource "netbox_config_context" "ntp_apac" {
  name       = "ntp-apac"
  profile_id = data.netbox_config_context_profile.ntp.id
  data       = jsonencode({ ntp_servers = ["10.20.0.1"] })
}
