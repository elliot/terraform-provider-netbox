data "netbox_ipam_role" "example" {
  slug = "production"
}

# Any API filter of /api/ipam/roles/ works; the lookup must match exactly one object.
data "netbox_ipam_role" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "ipam_role_id" {
  value = data.netbox_ipam_role.example.id
}
