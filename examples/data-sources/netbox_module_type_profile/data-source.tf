data "netbox_module_type_profile" "line_card" {
  name = "Line card"
}

output "line_card_schema" {
  value = jsondecode(data.netbox_module_type_profile.line_card.schema)
}
