data "netbox_provider_accounts" "lumen" {
  filters = [
    { name = "provider", value = "lumen" },
  ]
}

output "lumen_accounts" {
  value = data.netbox_provider_accounts.lumen.items[*].account
}
