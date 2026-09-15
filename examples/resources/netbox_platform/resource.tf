resource "netbox_manufacturer" "juniper" {
  name = "Juniper Networks"
  slug = "juniper"
}

resource "netbox_config_template" "junos_base" {
  name          = "Junos base"
  template_code = "system {\n  host-name {{ device.name }};\n}\n"
}

resource "netbox_platform" "junos" {
  name            = "Junos"
  slug            = "junos"
  manufacturer_id = netbox_manufacturer.juniper.id
  description     = "Juniper Junos OS"
}

# Platforms can nest (NetBox 4.4+), e.g. a specific release under the OS family.
resource "netbox_platform" "junos_evo" {
  name               = "Junos Evolved"
  slug               = "junos-evo"
  parent_id          = netbox_platform.junos.id
  manufacturer_id    = netbox_manufacturer.juniper.id
  config_template_id = netbox_config_template.junos_base.id
  description        = "Junos OS Evolved"
}
