# All vm interfaces (paginated by NetBox; use limit to cap the result).
data "netbox_vm_interfaces" "all" {}

# Filters take any query parameter of /api/virtualization/interfaces/.
data "netbox_vm_interfaces" "filtered" {
  filters = [
    { name = "virtual_machine", value = "web01.example.com" },
  ]
  limit = 50
}

output "vm_interfaces_ids" {
  value = data.netbox_vm_interfaces.filtered.items[*].id
}
