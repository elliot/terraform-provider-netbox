data "netbox_rir" "example" {
  slug = "rfc-1918"
}

# Any API filter of /api/ipam/rirs/ works; the lookup must match exactly one object.
data "netbox_rir" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "rir_id" {
  value = data.netbox_rir.example.id
}
