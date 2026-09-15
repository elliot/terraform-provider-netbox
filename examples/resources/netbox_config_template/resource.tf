# Jinja2 device configuration template; render it from the device page or the
# /api/dcim/devices/<id>/render-config/ endpoint.
resource "netbox_config_template" "base" {
  name        = "base-config"
  description = "Hostname, NTP and syslog from config context"
  template_code = <<-EOT
    hostname {{ device.name }}
    {% for server in ntp_servers %}
    ntp server {{ server }}
    {% endfor %}
    {% if syslog is defined %}
    logging host {{ syslog.host }}
    {% endif %}
  EOT
  environment_params = jsonencode({ trim_blocks = true, lstrip_blocks = true })
  mime_type          = "text/plain"
  file_extension     = "cfg"
  as_attachment      = true
}

resource "netbox_device_role" "access" {
  name               = "Access switch"
  slug               = "access-switch"
  config_template_id = netbox_config_template.base.id
}
