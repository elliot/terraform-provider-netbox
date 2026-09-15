data "netbox_custom_field_choice_sets" "iso" {
  filters = [{ name = "base_choices", value = "ISO_3166" }]
}

output "iso_choice_sets" {
  value = data.netbox_custom_field_choice_sets.iso.items[*].name
}
