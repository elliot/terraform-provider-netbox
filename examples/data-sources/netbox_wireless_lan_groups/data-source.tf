# Direct children of the "campus" group.
data "netbox_wireless_lan_groups" "campus_children" {
  filters = [{ name = "parent", value = "campus" }]
}

output "campus_child_groups" {
  value = data.netbox_wireless_lan_groups.campus_children.items[*].slug
}
