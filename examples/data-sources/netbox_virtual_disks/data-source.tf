# All virtual disks (paginated by NetBox; use limit to cap the result).
data "netbox_virtual_disks" "all" {}

# Filters take any query parameter of /api/virtualization/virtual-disks/.
data "netbox_virtual_disks" "filtered" {
  filters = [
    { name = "virtual_machine", value = "db01.example.com" },
  ]
  limit = 50
}

output "virtual_disks_ids" {
  value = data.netbox_virtual_disks.filtered.items[*].id
}
