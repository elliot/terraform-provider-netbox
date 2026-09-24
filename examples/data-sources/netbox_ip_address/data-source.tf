data "netbox_ip_address" "example" {
  address = "10.10.20.1/24"
}

# Any API filter of /api/ipam/ip-addresses/ works; the lookup must match exactly one object.
data "netbox_ip_address" "filtered" {
  filters = [
    { name = "dns_name", value = "gw.office.example.com" },
  ]
}

output "ip_address_id" {
  value = data.netbox_ip_address.example.id
}
