# Look up a single virtual disk by name.
data "netbox_virtual_disk" "example" {
  name = "root"
}

# Any API filter of /api/virtualization/virtual-disks/ works with filters; the lookup must match exactly one object.
data "netbox_virtual_disk" "filtered" {
  filters = [
    { name = "virtual_machine", value = "db01.example.com" },
  ]
}

output "virtual_disk_id" {
  value = data.netbox_virtual_disk.example.id
}
