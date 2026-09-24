data "netbox_vrf" "example" {
  name = "ACME-L3VPN"
}

# Any API filter of /api/ipam/vrfs/ works; the lookup must match exactly one object.
data "netbox_vrf" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "vrf_id" {
  value = data.netbox_vrf.example.id
}
