resource "netbox_inventory_item_role" "psu" {
  name        = "Power supply"
  slug        = "power-supply"
  color       = "ff9800"
  description = "Field-replaceable power supplies"
}
resource "netbox_inventory_item_role" "fan" {
  name  = "Fan"
  slug  = "fan"
  color = "2196f3"
}
