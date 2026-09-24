data "netbox_config_template" "base" {
  name = "base-config"
}

resource "netbox_platform" "ios" {
  name               = "Cisco IOS"
  slug               = "cisco-ios"
  config_template_id = data.netbox_config_template.base.id
}
