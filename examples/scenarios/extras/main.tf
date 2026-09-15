# Extras and wireless scenario
#
# Exercises the "extras" customisation layer of NetBox end to end: tags, a
# custom field backed by a choice set that is applied to sites and read back
# through the site data source, a config context validated by a profile and
# rendered into a device configuration template, a webhook fired by an event
# rule, a journal entry on the site, and a wireless LAN group with a guest
# SSID mapped to a VLAN.
#
# Validated against https://demo.netbox.dev with the locally built provider:
#   export NETBOX_SERVER_URL=https://demo.netbox.dev NETBOX_API_TOKEN=...
#   terraform init && terraform apply && terraform destroy
#
# After apply, the device's rendered configuration (template + config
# context) is available at
#   POST /api/dcim/devices/<device_id>/render-config/

terraform {
  required_providers {
    netbox = {
      source = "elliot/netbox"
    }
  }
}

provider "netbox" {
  # server_url and api_token come from NETBOX_SERVER_URL / NETBOX_API_TOKEN.
}

variable "prefix" {
  description = "Prefix applied to every object name and slug so the scenario can coexist with other data (and be swept)."
  type        = string
  default     = "tfacc-extras"
}

locals {
  # Custom field names must match ^[a-z0-9_]+$.
  cf_name = "${replace(var.prefix, "-", "_")}_support_tier"
}

# --------------------------------------------------------------------- tags

resource "netbox_tag" "managed" {
  name        = "${var.prefix} managed"
  slug        = "${var.prefix}-managed"
  color       = "2196f3"
  description = "Objects owned by the ${var.prefix} scenario"
}

resource "netbox_tag" "wireless" {
  name         = "${var.prefix} wireless"
  slug         = "${var.prefix}-wireless"
  color        = "9c27b0"
  object_types = ["wireless.wirelesslan", "wireless.wirelesslangroup"]
}

# ------------------------------------------------------------ custom fields

resource "netbox_custom_field_choice_set" "support_tier" {
  name        = "${var.prefix} support tier"
  description = "Colocation support contract level"
  # Sorted: order_alphabetically makes NetBox return the choices sorted.
  extra_choices = [
    ["bronze", "Bronze"],
    ["gold", "Gold"],
    ["silver", "Silver"],
  ]
  choice_colors        = jsonencode({ bronze = "orange", silver = "gray", gold = "yellow" })
  order_alphabetically = true
}

resource "netbox_custom_field" "support_tier" {
  name          = local.cf_name
  label         = "Support tier"
  group_name    = "Contract"
  description   = "${var.prefix} scenario"
  type          = "select"
  object_types  = ["dcim.site"]
  choice_set_id = netbox_custom_field_choice_set.support_tier.id
  default       = jsonencode("bronze")
  weight        = 100
}

# ------------------------------------------------------------------- site

resource "netbox_site" "ams" {
  name        = "${var.prefix} Amsterdam"
  slug        = "${var.prefix}-ams"
  status      = "active"
  description = "Colocation site managed by the ${var.prefix} scenario"
  tags        = [netbox_tag.managed.slug]

  # Selection values are written as their choice value; NetBox 4.7 returns
  # them as {value,label} objects, which the provider unwraps again.
  custom_fields = {
    (local.cf_name) = "gold"
  }

  depends_on = [netbox_custom_field.support_tier]
}

# Read the site back and extract the custom field through the data source,
# which exposes every custom field of the object.
data "netbox_site" "ams" {
  id = netbox_site.ams.id
}

# --------------------------------------------------------- config contexts

resource "netbox_config_context_profile" "ntp" {
  name        = "${var.prefix} ntp"
  description = "Schema for NTP and syslog settings"
  schema = jsonencode({
    type = "object"
    properties = {
      ntp_servers = { type = "array", items = { type = "string" } }
      syslog = {
        type       = "object"
        properties = { host = { type = "string" }, port = { type = "integer" } }
      }
    }
    required = ["ntp_servers"]
  })
  tags = [netbox_tag.managed.slug]
}

resource "netbox_config_context" "ams" {
  name        = "${var.prefix} ams"
  description = "Site-local NTP and syslog servers"
  profile_id  = netbox_config_context_profile.ntp.id
  weight      = 1000
  site_ids    = [netbox_site.ams.id]
  data = jsonencode({
    ntp_servers = ["10.10.0.1", "10.10.0.2"]
    syslog      = { host = "10.10.0.3", port = 514 }
  })
  tags = [netbox_tag.managed.slug]
}

# ---------------------------------------------------------- config template

resource "netbox_config_template" "base" {
  name        = "${var.prefix} base-config"
  description = "Hostname, NTP and syslog from the merged config context"
  # NetBox strips leading/trailing whitespace from text fields, so trim the
  # heredoc to keep the plan stable.
  template_code = trimspace(<<-EOT
    hostname {{ device.name }}
    {% for server in ntp_servers %}
    ntp server {{ server }}
    {% endfor %}
    {% if syslog is defined %}
    logging host {{ syslog.host }} port {{ syslog.port }}
    {% endif %}
  EOT
  )
  environment_params = jsonencode({ trim_blocks = true, lstrip_blocks = true })
  mime_type          = "text/plain"
  file_extension     = "cfg"
  as_attachment      = true
  tags               = [netbox_tag.managed.slug]
}

# A device at the site to render the template for.

resource "netbox_manufacturer" "generic" {
  name = "${var.prefix} Generic"
  slug = "${var.prefix}-generic"
}

resource "netbox_device_type" "access" {
  manufacturer_id = netbox_manufacturer.generic.id
  model           = "${var.prefix} access-48"
  slug            = "${var.prefix}-access-48"
  u_height        = 1
}

resource "netbox_device_role" "access" {
  name  = "${var.prefix} access switch"
  slug  = "${var.prefix}-access-switch"
  color = "4caf50"
}

resource "netbox_device" "sw1" {
  name               = "${var.prefix}-ams-sw1"
  device_type_id     = netbox_device_type.access.id
  role_id            = netbox_device_role.access.id
  site_id            = netbox_site.ams.id
  status             = "active"
  config_template_id = netbox_config_template.base.id
  tags               = [netbox_tag.managed.slug]
}

# ------------------------------------------------------- webhook + event rule

resource "netbox_webhook" "cmdb" {
  name              = "${var.prefix} cmdb-sync"
  description       = "Push device changes to the CMDB"
  payload_url       = "https://cmdb.example.com/api/netbox/${var.prefix}"
  http_method       = "POST"
  http_content_type = "application/json"
  body_template     = "{\"event\": \"{{ event }}\", \"model\": \"{{ model }}\", \"name\": \"{{ data.name }}\"}"
  ssl_verification  = true
  timeout           = 10
  tags              = [netbox_tag.managed.slug]
}

resource "netbox_event_rule" "device_changes" {
  name               = "${var.prefix} device-changes"
  description        = "Sync active devices to the CMDB"
  object_types       = ["dcim.device"]
  event_types        = ["object_created", "object_updated", "object_deleted"]
  action_type        = "webhook"
  action_object_type = "extras.webhook"
  action_object_id   = netbox_webhook.cmdb.id
  conditions         = jsonencode({ attr = "status.value", value = "active" })
  tags               = [netbox_tag.managed.slug]
}

# ----------------------------------------------------------- journal entry

resource "netbox_journal_entry" "commissioning" {
  assigned_object_type = "dcim.site"
  assigned_object_id   = netbox_site.ams.id
  kind                 = "info"
  comments             = "${var.prefix}: site commissioned by Terraform; support tier ${netbox_site.ams.custom_fields[local.cf_name]}"
}

# --------------------------------------------------------------- wireless

resource "netbox_wireless_lan_group" "campus" {
  name        = "${var.prefix} campus"
  slug        = "${var.prefix}-campus"
  description = "Campus SSIDs"
  tags        = [netbox_tag.wireless.slug]
}

resource "netbox_vlan" "guest" {
  name    = "${var.prefix}-guest"
  vid     = 1200
  site_id = netbox_site.ams.id
  status  = "active"
}

resource "netbox_wireless_lan" "guest" {
  ssid        = "${var.prefix}-Guest"
  description = "Captive-portal guest network"
  group_id    = netbox_wireless_lan_group.campus.id
  vlan_id     = netbox_vlan.guest.id
  status      = "active"
  auth_type   = "wpa-personal"
  auth_cipher = "aes"
  auth_psk    = "${var.prefix}-guest-psk"
  tags        = [netbox_tag.wireless.slug]
}

# ---------------------------------------------------------------- outputs

output "site_support_tier" {
  description = "Custom field value read back from the site data source."
  value       = jsondecode(data.netbox_site.ams.custom_fields)[local.cf_name]
}

output "device_id" {
  description = "Render the configuration with POST /api/dcim/devices/<device_id>/render-config/."
  value       = netbox_device.sw1.id
}

output "guest_ssid_vlan" {
  value = netbox_vlan.guest.vid
}
