# Every virtual chassis of one parent object.
data "netbox_virtual_chassis_list" "all" {
  filters = [
    { name = "domain", value = "stack01" },
  ]
}
output "virtual_chassis_list_names" {
  value = data.netbox_virtual_chassis_list.all.items[*].name
}
