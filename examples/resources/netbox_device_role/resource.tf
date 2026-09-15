resource "netbox_config_template" "switch_base" {
  name          = "Switch base"
  template_code = "hostname {{ device.name }}\n"
}

resource "netbox_device_role" "switch" {
  name        = "Switch"
  slug        = "switch"
  color       = "2196f3"
  description = "Any Ethernet switch"
}

# Device roles nest (NetBox 4.3+); vm_role controls whether the role can be
# assigned to virtual machines.
resource "netbox_device_role" "access_switch" {
  name               = "Access switch"
  slug               = "access-switch"
  color              = "4caf50"
  vm_role            = false
  parent_id          = netbox_device_role.switch.id
  config_template_id = netbox_config_template.switch_base.id
  description        = "Top-of-rack access switch"
}
