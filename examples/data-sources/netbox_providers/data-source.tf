# Providers that have at least one circuit.
data "netbox_providers" "active" {
  filters = [
    { name = "q", value = "Lumen" },
  ]
  limit = 50
}

output "provider_slugs" {
  value = data.netbox_providers.active.items[*].slug
}
