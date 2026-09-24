data "netbox_virtual_chassis" "stack1" {
  name = "dc1-stack01"
}
output "stack_master" {
  value = data.netbox_virtual_chassis.stack1.master_id
}
