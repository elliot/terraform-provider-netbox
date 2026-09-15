# A selection field backed by a choice set, applied to sites and devices.
resource "netbox_custom_field_choice_set" "support_tier" {
  name          = "Support tier"
  extra_choices = [["bronze", "Bronze"], ["silver", "Silver"], ["gold", "Gold"]]
}

resource "netbox_custom_field" "support_tier" {
  name          = "support_tier"
  label         = "Support tier"
  group_name    = "Contract"
  type          = "select"
  object_types  = ["dcim.site", "dcim.device"]
  choice_set_id = netbox_custom_field_choice_set.support_tier.id
  default       = jsonencode("bronze")
  filter_logic  = "exact"
  weight        = 100
}

# A validated integer field.
resource "netbox_custom_field" "rack_units" {
  name               = "contracted_rack_units"
  label              = "Contracted rack units"
  type               = "integer"
  object_types       = ["dcim.site"]
  validation_minimum = 0
  validation_maximum = 10000
  description        = "Rack units contracted with the colocation provider"
}

# Values are set through the custom_fields attribute of the target object
# (or with netbox_custom_field_value for objects managed elsewhere).
resource "netbox_site" "example" {
  name = "Amsterdam"
  slug = "ams"
  custom_fields = {
    support_tier          = "gold"
    contracted_rack_units = 42
  }
  depends_on = [netbox_custom_field.support_tier, netbox_custom_field.rack_units]
}
