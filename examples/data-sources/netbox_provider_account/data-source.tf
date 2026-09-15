data "netbox_provider" "lumen" {
  slug = "lumen"
}

# Accounts are unique per provider, so filter by provider as well as name.
data "netbox_provider_account" "billing" {
  filters = [
    { name = "provider_id", value = tostring(data.netbox_provider.lumen.id) },
    { name = "account", value = "ACCT-00812345" },
  ]
}
