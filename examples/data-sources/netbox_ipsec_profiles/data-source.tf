data "netbox_ipsec_profiles" "all" {}

data "netbox_ipsec_profiles" "filtered" {
  filters = [
    { name = "mode", value = "esp" },
  ]
  limit = 50
}

output "ipsec_profiles_names" {
  value = data.netbox_ipsec_profiles.filtered.items[*].name
}
