# Every module type of one parent object.
data "netbox_module_types" "all" {
  filters = [
    { name = "manufacturer", value = "cisco" },
  ]
}
output "module_types_names" {
  value = data.netbox_module_types.all.items[*].name
}
