data "netbox_fhrp_group_assignments" "all" {}

data "netbox_fhrp_group_assignments" "filtered" {
  filters = [
    { name = "interface_type", value = "dcim.interface" },
  ]
  limit = 50
}

output "fhrp_group_assignments_ids" {
  value = data.netbox_fhrp_group_assignments.filtered.items[*].id
}
