# Look up a single vm interface by name.
data "netbox_vm_interface" "example" {
  name = "eth0"
}

# Any API filter of /api/virtualization/interfaces/ works with filters; the lookup must match exactly one object.
data "netbox_vm_interface" "filtered" {
  filters = [
    { name = "virtual_machine", value = "web01.example.com" },
  ]
}

output "vm_interface_id" {
  value = data.netbox_vm_interface.example.id
}
