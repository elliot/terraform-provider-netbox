resource "netbox_provider" "example" {
  name = "Lumen"
  slug = "lumen"
}

resource "netbox_provider_account" "example" {
  provider_id = netbox_provider.example.id
  account     = "ACCT-00812345"
  name        = "EMEA billing"
  description = "Master services agreement for the European sites"
}
