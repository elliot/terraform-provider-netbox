data "netbox_asn" "example" {
  asn = 4200000001
}

# Any API filter of /api/ipam/asns/ works; the lookup must match exactly one object.
data "netbox_asn" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "asn_id" {
  value = data.netbox_asn.example.id
}
