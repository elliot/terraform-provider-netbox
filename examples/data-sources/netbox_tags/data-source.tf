# Tags whose name or slug contains "managed". Note: object_types /
# for_object_type_id take numeric content-type IDs, not "app.model" labels.
data "netbox_tags" "managed" {
  filters = [{ name = "q", value = "managed" }]
}

output "managed_tag_slugs" {
  value = data.netbox_tags.managed.items[*].slug
}
