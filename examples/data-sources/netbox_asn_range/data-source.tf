data "netbox_asn_range" "example" {
  slug = "fabric-underlay"
}

# Any API filter of /api/ipam/asn-ranges/ works; the lookup must match exactly one object.
data "netbox_asn_range" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "asn_range_id" {
  value = data.netbox_asn_range.example.id
}
