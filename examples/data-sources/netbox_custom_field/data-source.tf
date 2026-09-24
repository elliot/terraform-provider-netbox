data "netbox_custom_field" "support_tier" {
  name = "support_tier"
}

output "support_tier_choice_set" {
  value = data.netbox_custom_field.support_tier.choice_set_id
}
