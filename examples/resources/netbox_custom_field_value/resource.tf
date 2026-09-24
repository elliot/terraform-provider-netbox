# Annotate an object that this configuration does not manage: here an
# existing device found through a data source.
data "netbox_device" "core1" {
  name = "dc1-core-01"
}

resource "netbox_custom_field_value" "core1_owner" {
  object_type = "dcim.device"
  object_id   = data.netbox_device.core1.id
  name        = "business_owner"
  value       = "network-platform-team"
}

# Object-typed custom fields take the related object's ID; multi-value
# fields take a list.
resource "netbox_custom_field_value" "core1_backup_sites" {
  object_type = "dcim.device"
  object_id   = data.netbox_device.core1.id
  name        = "backup_sites"
  value       = [12, 15]
}
