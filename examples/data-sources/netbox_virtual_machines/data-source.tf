# All virtual machines (paginated by NetBox; use limit to cap the result).
data "netbox_virtual_machines" "all" {}

# Filters take any query parameter of /api/virtualization/virtual-machines/.
data "netbox_virtual_machines" "filtered" {
  filters = [
    { name = "cluster", value = "fra1-prod-01" },
  ]
  limit = 50
}

output "virtual_machines_ids" {
  value = data.netbox_virtual_machines.filtered.items[*].id
}
