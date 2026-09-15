# Owner groups (NetBox 4.7+) organise owners, the teams accountable for objects.
resource "netbox_owner_group" "infrastructure" {
  name        = "Infrastructure"
  description = "Teams operating the physical and virtual infrastructure"
}
