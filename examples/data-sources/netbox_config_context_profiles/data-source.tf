data "netbox_config_context_profiles" "all" {}

output "profile_names" {
  value = data.netbox_config_context_profiles.all.items[*].name
}
