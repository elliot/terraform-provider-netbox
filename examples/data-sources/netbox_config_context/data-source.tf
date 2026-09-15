data "netbox_config_context" "ntp_emea" {
  name = "ntp-emea"
}

output "emea_ntp_servers" {
  value = jsondecode(data.netbox_config_context.ntp_emea.data).ntp_servers
}
