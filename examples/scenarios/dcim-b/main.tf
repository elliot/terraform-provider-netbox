# Scenario: dcim part B (device components, templates, modules, cables, VC).
#
# Builds a small access layer on one site:
#   * a switch device type carrying interface, console and power port
#     templates (instantiated automatically on every device of that type),
#   * a patch panel with a 12-strand MPO rear port fanned out to LC front ports,
#   * two switches stacked in a virtual chassis, each with explicit uplink
#     interfaces, a console server port, a module bay and an installed uplink
#     module whose interface templates are replicated onto the device,
#   * a cable between the two switches and a cable from a switch to the panel,
#   * an inventory item and a virtual device context on the first switch.
#
# Run against https://demo.netbox.dev with the locally built provider:
#   export TF_CLI_CONFIG_FILE=~/.terraformrc-dcim-b   # dev_overrides -> /tmp/tfp-dcim-b
#   export NETBOX_SERVER_URL=... NETBOX_API_TOKEN=... NETBOX_REQUESTS_PER_SECOND=2
#   terraform apply && terraform destroy

terraform {
  required_providers {
    netbox = {
      source = "elliot/netbox"
    }
  }
}

provider "netbox" {}

variable "prefix" {
  description = "Prefix for every object name/slug so the run is easy to find and sweep."
  type        = string
  default     = "tfacc-dcim-b"
}

# ---------------------------------------------------------------------------
# Site, roles, manufacturer
# ---------------------------------------------------------------------------
resource "netbox_site" "dc1" {
  name   = "${var.prefix}-dc1"
  slug   = "${var.prefix}-dc1"
  status = "active"
}

resource "netbox_device_role" "access_switch" {
  name  = "${var.prefix}-access-switch"
  slug  = "${var.prefix}-access-switch"
  color = "2196f3"
}

resource "netbox_device_role" "patch_panel" {
  name  = "${var.prefix}-patch-panel"
  slug  = "${var.prefix}-patch-panel"
  color = "9e9e9e"
}

resource "netbox_manufacturer" "vendor" {
  name = "${var.prefix}-vendor"
  slug = "${var.prefix}-vendor"
}

# ---------------------------------------------------------------------------
# Switch device type with component templates
# ---------------------------------------------------------------------------
resource "netbox_device_type" "switch" {
  manufacturer_id = netbox_manufacturer.vendor.id
  model           = "${var.prefix}-switch-48p"
  slug            = "${var.prefix}-switch-48p"
  part_number     = "SW-48P"
  u_height        = 1
  airflow         = "front-to-rear"
  description     = "48-port access switch (scenario ${var.prefix})"
}

resource "netbox_interface_template" "access" {
  count          = 4
  device_type_id = netbox_device_type.switch.id
  name           = "GigabitEthernet1/0/${count.index + 1}"
  label          = "Gi1/0/${count.index + 1}"
  type           = "1000base-t"
  poe_mode       = "pse"
  poe_type       = "type2-ieee802.3at"
}

resource "netbox_interface_template" "mgmt" {
  device_type_id = netbox_device_type.switch.id
  name           = "GigabitEthernet0/0"
  type           = "1000base-t"
  mgmt_only      = true
  description    = "Out-of-band management"
}

resource "netbox_console_port_template" "console" {
  device_type_id = netbox_device_type.switch.id
  name           = "Console"
  label          = "CON"
  type           = "rj-45"
}

resource "netbox_power_port_template" "psu" {
  count          = 2
  device_type_id = netbox_device_type.switch.id
  name           = "PSU${count.index + 1}"
  type           = "iec-60320-c14"
  maximum_draw   = 715
  allocated_draw = 350
}

resource "netbox_module_bay_template" "nm" {
  device_type_id = netbox_device_type.switch.id
  name           = "Network Module 1"
  label          = "NM1"
  position       = "1"
}

resource "netbox_inventory_item_role" "psu" {
  name  = "${var.prefix}-psu"
  slug  = "${var.prefix}-psu"
  color = "ff9800"
}

resource "netbox_inventory_item_template" "psu" {
  count           = 2
  device_type_id  = netbox_device_type.switch.id
  name            = "PSU ${count.index + 1}"
  role_id         = netbox_inventory_item_role.psu.id
  manufacturer_id = netbox_manufacturer.vendor.id
  part_id         = "PWR-715WAC"
}

# ---------------------------------------------------------------------------
# Uplink module type (its interface templates are replicated when installed)
# ---------------------------------------------------------------------------
resource "netbox_module_type" "nm_8x" {
  manufacturer_id = netbox_manufacturer.vendor.id
  model           = "${var.prefix}-nm-8x"
  part_number     = "NM-8X"
  description     = "8x 10G SFP+ uplink module"
}

resource "netbox_interface_template" "nm_uplinks" {
  count          = 8
  module_type_id = netbox_module_type.nm_8x.id
  name           = "TenGigabitEthernet1/{module}/${count.index + 1}"
  type           = "10gbase-x-sfpp"
}

# ---------------------------------------------------------------------------
# Patch panel device type (ports are managed explicitly on the device below)
# ---------------------------------------------------------------------------
resource "netbox_device_type" "panel" {
  manufacturer_id = netbox_manufacturer.vendor.id
  model           = "${var.prefix}-panel-24lc"
  slug            = "${var.prefix}-panel-24lc"
  u_height        = 1
}

# ---------------------------------------------------------------------------
# Virtual chassis and its two member switches
# ---------------------------------------------------------------------------
resource "netbox_virtual_chassis" "stack1" {
  name        = "${var.prefix}-stack01"
  domain      = "stack01"
  description = "Two-member stack (scenario ${var.prefix})"
}

resource "netbox_device" "sw" {
  count              = 2
  name               = "${var.prefix}-sw0${count.index + 1}"
  device_type_id     = netbox_device_type.switch.id
  role_id            = netbox_device_role.access_switch.id
  site_id            = netbox_site.dc1.id
  status             = "active"
  serial             = "${upper(var.prefix)}-SW0${count.index + 1}"
  virtual_chassis_id = netbox_virtual_chassis.stack1.id
  vc_position        = count.index + 1
  vc_priority        = count.index == 0 ? 15 : 10

  # Make sure the templates exist before the devices are instantiated so the
  # switches are born with their interfaces, console and power ports.
  depends_on = [
    netbox_interface_template.access,
    netbox_interface_template.mgmt,
    netbox_console_port_template.console,
    netbox_power_port_template.psu,
    netbox_module_bay_template.nm,
    netbox_inventory_item_template.psu,
  ]
}

resource "netbox_device" "pp01" {
  name           = "${var.prefix}-pp01"
  device_type_id = netbox_device_type.panel.id
  role_id        = netbox_device_role.patch_panel.id
  site_id        = netbox_site.dc1.id
}

# Components auto-created from the templates can be read back and used by
# other resources.
data "netbox_interface" "sw01_gi1_0_1" {
  filters = [
    { name = "device_id", value = netbox_device.sw[0].id },
    { name = "name", value = "GigabitEthernet1/0/1" },
  ]
}

# ---------------------------------------------------------------------------
# Explicit components on the switches
# ---------------------------------------------------------------------------
resource "netbox_interface" "uplink" {
  count       = 2
  device_id   = netbox_device.sw[count.index].id
  name        = "GigabitEthernet1/1/1"
  label       = "Gi1/1/1"
  type        = "1000base-x-sfp"
  description = "Stack peer link"
}

resource "netbox_interface" "to_panel" {
  device_id   = netbox_device.sw[0].id
  name        = "GigabitEthernet1/1/2"
  type        = "1000base-x-sfp"
  description = "To patch panel ${netbox_device.pp01.name}"
}

resource "netbox_console_server_port" "aux" {
  device_id = netbox_device.sw[0].id
  name      = "AUX"
  type      = "rj-45"
  speed     = 9600
}

resource "netbox_power_port" "psu3" {
  device_id    = netbox_device.sw[0].id
  name         = "PSU3"
  type         = "iec-60320-c14"
  maximum_draw = 715
  description  = "Redundant supply added in the field"
}

# A second module bay next to the templated one, with an installed module.
resource "netbox_module_bay" "nm2" {
  count     = 2
  device_id = netbox_device.sw[count.index].id
  name      = "Network Module 2"
  label     = "NM2"
  position  = "2"
}

resource "netbox_module" "nm2" {
  count          = 2
  device_id      = netbox_device.sw[count.index].id
  module_bay_id  = netbox_module_bay.nm2[count.index].id
  module_type_id = netbox_module_type.nm_8x.id
  status         = "active"
  serial         = "${upper(var.prefix)}-NM-${count.index + 1}"

  replicate_components = true
}

resource "netbox_inventory_item" "sfp" {
  device_id      = netbox_device.sw[0].id
  name           = "SFP-1G-LX in Gi1/1/2"
  part_id        = "SFP-1G-LX"
  serial         = "${upper(var.prefix)}-SFP-1"
  status         = "active"
  component_type = "dcim.interface"
  component_id   = netbox_interface.to_panel.id
}

resource "netbox_virtual_device_context" "tenant_a" {
  device_id   = netbox_device.sw[0].id
  name        = "${var.prefix}-vdc-a"
  identifier  = 10
  status      = "active"
  description = "Customer A context"
}

# ---------------------------------------------------------------------------
# Patch panel ports: one MPO trunk on the back, LC ports on the front
# ---------------------------------------------------------------------------
resource "netbox_rear_port" "trunk1" {
  device_id = netbox_device.pp01.id
  name      = "Trunk 1"
  label     = "T1"
  type      = "mpo"
  positions = 12
  color     = "00ffff"
}

resource "netbox_front_port" "lc" {
  count     = 4
  device_id = netbox_device.pp01.id
  name      = "LC ${count.index + 1}"
  label     = tostring(count.index + 1)
  type      = "lc"
  rear_ports = [
    {
      position           = 1
      rear_port          = netbox_rear_port.trunk1.id
      rear_port_position = count.index + 1
    },
  ]
}

# ---------------------------------------------------------------------------
# Cables
# ---------------------------------------------------------------------------
resource "netbox_cable" "stack_link" {
  a_terminations = [{ object_type = "dcim.interface", object_id = netbox_interface.uplink[0].id }]
  b_terminations = [{ object_type = "dcim.interface", object_id = netbox_interface.uplink[1].id }]
  type           = "smf-os2"
  status         = "connected"
  label          = "${var.prefix}-0001"
  color          = "ffeb3b"
  length         = 2
  length_unit    = "m"
}

resource "netbox_cable" "to_panel" {
  a_terminations = [{ object_type = "dcim.interface", object_id = netbox_interface.to_panel.id }]
  b_terminations = [{ object_type = "dcim.frontport", object_id = netbox_front_port.lc[0].id }]
  type           = "smf-os2"
  status         = "connected"
  label          = "${var.prefix}-0002"
  length         = 5
  length_unit    = "m"
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------
output "stack_members" {
  value = { for d in netbox_device.sw : d.name => d.vc_position }
}

output "templated_interface_id" {
  description = "Interface created by NetBox from the device type template."
  value       = data.netbox_interface.sw01_gi1_0_1.id
}

output "cable_ids" {
  value = [netbox_cable.stack_link.id, netbox_cable.to_panel.id]
}
