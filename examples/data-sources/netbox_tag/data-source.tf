data "netbox_tag" "managed" {
  slug = "managed-by-terraform"
}

output "managed_tag_color" {
  value = data.netbox_tag.managed.color
}
