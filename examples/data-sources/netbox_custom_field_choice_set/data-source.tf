data "netbox_custom_field_choice_set" "support_tier" {
  name = "Support tier"
}

# Reuse an existing choice set for a new field.
resource "netbox_custom_field" "vendor_tier" {
  name          = "vendor_tier"
  type          = "select"
  object_types  = ["dcim.manufacturer"]
  choice_set_id = data.netbox_custom_field_choice_set.support_tier.id
}
