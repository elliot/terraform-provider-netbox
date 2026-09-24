# Look up a single virtual machine type by slug.
data "netbox_virtual_machine_type" "example" {
  slug = "m-large"
}

# Any API filter of /api/virtualization/virtual-machine-types/ works with filters; the lookup must match exactly one object.
data "netbox_virtual_machine_type" "filtered" {
  filters = [
    { name = "q", value = "large" },
  ]
}

output "virtual_machine_type_id" {
  value = data.netbox_virtual_machine_type.example.id
}
