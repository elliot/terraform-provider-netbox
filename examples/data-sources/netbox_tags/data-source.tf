# Tags applicable to devices.
data "netbox_tags" "device" {
  filters = [{ name = "for_object_type", value = "dcim.device" }]
}

output "device_tag_slugs" {
  value = data.netbox_tags.device.items[*].slug
}
