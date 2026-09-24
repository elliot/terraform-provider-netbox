# Look up a single virtual machine by name.
data "netbox_virtual_machine" "example" {
  name = "web01.example.com"
}

# Any API filter of /api/virtualization/virtual-machines/ works with filters; the lookup must match exactly one object.
data "netbox_virtual_machine" "filtered" {
  filters = [
    { name = "cluster", value = "fra1-prod-01" },
  ]
}

output "virtual_machine_id" {
  value = data.netbox_virtual_machine.example.id
}
