# All virtual machine types (paginated by NetBox; use limit to cap the result).
data "netbox_virtual_machine_types" "all" {}

# Filters take any query parameter of /api/virtualization/virtual-machine-types/.
data "netbox_virtual_machine_types" "filtered" {
  filters = [
    { name = "q", value = "large" },
  ]
  limit = 50
}

output "virtual_machine_types_ids" {
  value = data.netbox_virtual_machine_types.filtered.items[*].id
}
