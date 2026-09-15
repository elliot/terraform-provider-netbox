resource "netbox_provider" "example" {
  name = "Lumen"
  slug = "lumen"
}

resource "netbox_provider_account" "example" {
  provider_id = netbox_provider.example.id
  account     = "ACCT-00812345"
}

resource "netbox_circuit_type" "example" {
  name = "Internet access"
  slug = "internet-access"
}

resource "netbox_tenant" "example" {
  name = "Acme Corp"
  slug = "acme-corp"
}

resource "netbox_circuit" "example" {
  cid                 = "LUM-DIA-48213"
  provider_id         = netbox_provider.example.id
  provider_account_id = netbox_provider_account.example.id
  type_id             = netbox_circuit_type.example.id
  tenant_id           = netbox_tenant.example.id
  status              = "active"
  install_date        = "2024-03-01"
  commit_rate         = 1000000 # Kbps
  description         = "1G DIA for the Amsterdam office"
}
