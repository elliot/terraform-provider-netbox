# Each choice is a [value, label] pair.
resource "netbox_custom_field_choice_set" "support_tier" {
  name        = "Support tier"
  description = "Vendor support contract level"
  extra_choices = [
    ["bronze", "Bronze"],
    ["silver", "Silver"],
    ["gold", "Gold"],
  ]
  # Colour names, not hex codes: blue, indigo, purple, pink, red, orange,
  # yellow, green, teal, cyan, gray, black, white.
  choice_colors = jsonencode({
    bronze = "orange"
    silver = "gray"
    gold   = "yellow"
  })
}

# Extend one of NetBox's built-in choice lists. With order_alphabetically the
# API returns extra_choices sorted, so keep the configured list sorted too.
resource "netbox_custom_field_choice_set" "countries" {
  name         = "Countries"
  base_choices = "ISO_3166"
  extra_choices = [
    ["XK", "Kosovo"],
  ]
  order_alphabetically = true
}
