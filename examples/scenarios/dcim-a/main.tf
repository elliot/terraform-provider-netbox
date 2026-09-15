# Scenario "dcim-a": a small data-centre footprint built from the DCIM
# facility, power and cooling resources.
#
#   region -> site -> location -> rack (with rack type, role, reservation)
#   power panel -> power feeds (A/B) -> rack
#   cooling source -> cooling feed -> rack
#   manufacturer -> device type / platform / device role -> device
#   device -> power port / outlet, cooling intake / outflow, interface + MAC
#
# Provider configuration comes from NETBOX_SERVER_URL and NETBOX_API_TOKEN.
# Every object carries the "tfacc-dcim-a" prefix so the demo sweeper can find
# it. Apply, inspect, then destroy:
#
#   terraform init && terraform apply && terraform destroy

terraform {
  required_providers {
    netbox = {
      source = "elliot/netbox"
    }
  }
}

provider "netbox" {}

locals {
  prefix = "tfacc-dcim-a"
}

# --- Facility hierarchy -----------------------------------------------------

resource "netbox_tag" "scenario" {
  name        = "${local.prefix}-scenario"
  slug        = "${local.prefix}-scenario"
  color       = "607d8b"
  description = "Objects created by the dcim-a Terraform scenario"
}

resource "netbox_tenant" "acme" {
  name        = "${local.prefix} ACME"
  slug        = "${local.prefix}-acme"
  description = "${local.prefix} demo tenant"
}

resource "netbox_region" "europe" {
  name        = "${local.prefix} Europe"
  slug        = "${local.prefix}-europe"
  description = "${local.prefix} parent region"
}

resource "netbox_region" "germany" {
  name        = "${local.prefix} Germany"
  slug        = "${local.prefix}-de"
  parent_id   = netbox_region.europe.id
  description = "${local.prefix} child region"
}

resource "netbox_site_group" "datacenters" {
  name        = "${local.prefix} Data centres"
  slug        = "${local.prefix}-datacenters"
  description = "${local.prefix} site group"
}

resource "netbox_site" "fra1" {
  name             = "${local.prefix} FRA1"
  slug             = "${local.prefix}-fra1"
  status           = "active"
  region_id        = netbox_region.germany.id
  group_id         = netbox_site_group.datacenters.id
  tenant_id        = netbox_tenant.acme.id
  facility         = "Interxion FRA1"
  time_zone        = "Europe/Berlin"
  description      = "${local.prefix} primary data centre"
  physical_address = "Hanauer Landstraße 300, 60314 Frankfurt am Main, Germany"
  latitude         = 50.114500
  longitude        = 8.735500
  tags             = [netbox_tag.scenario.slug]
}

resource "netbox_location" "floor_2" {
  name        = "${local.prefix} Floor 2"
  slug        = "${local.prefix}-fra1-floor-2"
  site_id     = netbox_site.fra1.id
  status      = "active"
  description = "${local.prefix} data hall"
  tags        = [netbox_tag.scenario.slug]
}

resource "netbox_location" "cage_a" {
  name        = "${local.prefix} Cage A"
  slug        = "${local.prefix}-fra1-cage-a"
  site_id     = netbox_site.fra1.id
  parent_id   = netbox_location.floor_2.id
  tenant_id   = netbox_tenant.acme.id
  status      = "active"
  facility    = "Cage 2A"
  description = "${local.prefix} private cage"
}

# --- Racks ------------------------------------------------------------------

resource "netbox_manufacturer" "apc" {
  name        = "${local.prefix} APC"
  slug        = "${local.prefix}-apc"
  description = "${local.prefix} rack manufacturer"
}

resource "netbox_rack_type" "netshelter" {
  manufacturer_id = netbox_manufacturer.apc.id
  model           = "${local.prefix} NetShelter SX 42U"
  slug            = "${local.prefix}-netshelter-sx-42u"
  form_factor     = "4-post-cabinet"
  width           = 19
  u_height        = 42
  outer_width     = 600
  outer_depth     = 1070
  outer_unit      = "mm"
  weight          = 122.5
  max_weight      = 1363
  weight_unit     = "kg"
  mounting_depth  = 900
  description     = "${local.prefix} rack type"
}

resource "netbox_rack_role" "compute" {
  name        = "${local.prefix} Compute"
  slug        = "${local.prefix}-compute"
  color       = "4caf50"
  description = "${local.prefix} rack role"
}

resource "netbox_rack_group" "compute" {
  name        = "${local.prefix} Compute racks"
  slug        = "${local.prefix}-compute-racks"
  description = "${local.prefix} rack group"
}

resource "netbox_rack" "a01" {
  name         = "${local.prefix} A01"
  site_id      = netbox_site.fra1.id
  location_id  = netbox_location.cage_a.id
  tenant_id    = netbox_tenant.acme.id
  role_id      = netbox_rack_role.compute.id
  group_id     = netbox_rack_group.compute.id
  rack_type_id = netbox_rack_type.netshelter.id
  status       = "active"
  facility_id  = "${local.prefix}-2A-01"
  serial       = "${local.prefix}-RK-00017"
  airflow      = "front-to-rear"
  description  = "${local.prefix} first compute rack"
  tags         = [netbox_tag.scenario.slug]
}

resource "netbox_user" "installer" {
  username = "${local.prefix}-installer"
  password = "${local.prefix}-Passw0rd!"
}

resource "netbox_rack_reservation" "a01_storage" {
  rack_id     = netbox_rack.a01.id
  user_id     = netbox_user.installer.id
  tenant_id   = netbox_tenant.acme.id
  units       = [1, 2, 3, 4]
  status      = "active"
  description = "${local.prefix} reserved for storage array"
}

# --- Power ------------------------------------------------------------------

resource "netbox_power_panel" "pp_a" {
  site_id     = netbox_site.fra1.id
  location_id = netbox_location.floor_2.id
  name        = "${local.prefix} PP-A"
  description = "${local.prefix} distribution panel A"
  tags        = [netbox_tag.scenario.slug]
}

resource "netbox_power_feed" "a01_a" {
  power_panel_id  = netbox_power_panel.pp_a.id
  rack_id         = netbox_rack.a01.id
  name            = "${local.prefix} A01-A"
  status          = "active"
  type            = "primary"
  supply          = "ac"
  phase           = "single-phase"
  voltage         = 230
  amperage        = 32
  max_utilization = 80
  tenant_id       = netbox_tenant.acme.id
  description     = "${local.prefix} primary feed"
}

resource "netbox_power_feed" "a01_b" {
  power_panel_id = netbox_power_panel.pp_a.id
  rack_id        = netbox_rack.a01.id
  name           = "${local.prefix} A01-B"
  type           = "redundant"
  voltage        = 230
  amperage       = 32
  description    = "${local.prefix} redundant feed"
}

# --- Cooling ----------------------------------------------------------------

resource "netbox_cooling_source" "chiller" {
  site_id          = netbox_site.fra1.id
  location_id      = netbox_location.floor_2.id
  name             = "${local.prefix} CH-01"
  type             = "chiller"
  status           = "active"
  fluid_type       = "water-glycol"
  cooling_capacity = 250
  description      = "${local.prefix} chiller"
  tags             = [netbox_tag.scenario.slug]
}

resource "netbox_cooling_feed" "a01" {
  cooling_source_id = netbox_cooling_source.chiller.id
  rack_id           = netbox_rack.a01.id
  name              = "${local.prefix} CH-01/A01"
  status            = "active"
  cooling_capacity  = 30
  max_flow          = 12.5
  max_flow_unit     = "lpm"
  description       = "${local.prefix} loop to rack A01"
}

# --- Device model -----------------------------------------------------------

resource "netbox_manufacturer" "example" {
  name        = "${local.prefix} Example Networks"
  slug        = "${local.prefix}-example-networks"
  description = "${local.prefix} device manufacturer"
}

resource "netbox_platform" "exos" {
  name            = "${local.prefix} ExOS"
  slug            = "${local.prefix}-exos"
  manufacturer_id = netbox_manufacturer.example.id
  description     = "${local.prefix} platform"
}

resource "netbox_device_role" "switch" {
  name        = "${local.prefix} Switch"
  slug        = "${local.prefix}-switch"
  color       = "2196f3"
  description = "${local.prefix} parent role"
}

resource "netbox_device_role" "access_switch" {
  name        = "${local.prefix} Access switch"
  slug        = "${local.prefix}-access-switch"
  color       = "4caf50"
  vm_role     = false
  parent_id   = netbox_device_role.switch.id
  description = "${local.prefix} child role"
}

resource "netbox_module_type_profile" "line_card" {
  name        = "${local.prefix} Line card"
  description = "${local.prefix} module type profile"
  schema = jsonencode({
    type = "object"
    properties = {
      ports = { type = "integer", title = "Port count" }
    }
  })
}

resource "netbox_module_bay_type" "sfp28" {
  name            = "${local.prefix} SFP28 cage"
  slug            = "${local.prefix}-sfp28-cage"
  manufacturer_id = netbox_manufacturer.example.id
  color           = "9c27b0"
  description     = "${local.prefix} module bay type"
}

resource "netbox_device_type" "sw48" {
  manufacturer_id = netbox_manufacturer.example.id
  model           = "${local.prefix} SW-48"
  slug            = "${local.prefix}-sw-48"
  u_height        = 1
  description     = "${local.prefix} device type"
}

resource "netbox_power_port_template" "psu1" {
  device_type_id = netbox_device_type.sw48.id
  name           = "PSU1"
  type           = "iec-60320-c14"
  maximum_draw   = 500
  allocated_draw = 350
  description    = "${local.prefix} power port template"
}

resource "netbox_power_outlet_template" "outlet_1" {
  device_type_id = netbox_device_type.sw48.id
  name           = "Outlet 1"
  type           = "iec-60320-c13"
  power_port_id  = netbox_power_port_template.psu1.id
  feed_leg       = "A"
  description    = "${local.prefix} power outlet template"
}

resource "netbox_cooling_intake_template" "in1" {
  device_type_id = netbox_device_type.sw48.id
  name           = "Intake 1"
  type           = "uqd"
  diameter       = 12.7
  diameter_unit  = "mm"
  description    = "${local.prefix} cooling intake template"
}

resource "netbox_cooling_outflow_template" "out1" {
  device_type_id    = netbox_device_type.sw48.id
  name              = "Outflow 1"
  type              = "uqdb"
  cooling_intake_id = netbox_cooling_intake_template.in1.id
  description       = "${local.prefix} cooling outflow template"
}

# --- A device in the rack ---------------------------------------------------

resource "netbox_device" "sw01" {
  name           = "${local.prefix}-fra1-sw01"
  device_type_id = netbox_device_type.sw48.id
  role_id        = netbox_device_role.access_switch.id
  platform_id    = netbox_platform.exos.id
  site_id        = netbox_site.fra1.id
  location_id    = netbox_location.cage_a.id
  rack_id        = netbox_rack.a01.id
  position       = 40
  face           = "front"
  tenant_id      = netbox_tenant.acme.id
  description    = "${local.prefix} top-of-rack switch"
  tags           = [netbox_tag.scenario.slug]
}

# Components instantiated from the device type templates are created by NetBox
# when the device is created (PSU1, Outlet 1, Intake 1, Outflow 1); the
# components below are additional ones managed explicitly.
resource "netbox_power_port" "psu2" {
  device_id      = netbox_device.sw01.id
  name           = "PSU2"
  type           = "iec-60320-c14"
  maximum_draw   = 500
  allocated_draw = 350
  description    = "${local.prefix} second power supply"
}

resource "netbox_power_outlet" "outlet_2" {
  device_id     = netbox_device.sw01.id
  name          = "Outlet 2"
  type          = "iec-60320-c13"
  power_port_id = netbox_power_port.psu2.id
  feed_leg      = "B"
  description   = "${local.prefix} outlet fed by PSU2"
}

resource "netbox_cooling_outflow" "out2" {
  device_id   = netbox_device.sw01.id
  name        = "Outflow 2"
  type        = "uqdb"
  description = "${local.prefix} second coolant return"
}

resource "netbox_cooling_intake" "in2" {
  device_id          = netbox_device.sw01.id
  name               = "Intake 2"
  type               = "uqd"
  diameter           = 12.7
  diameter_unit      = "mm"
  max_flow           = 4.5
  max_flow_unit      = "lpm"
  cooling_outflow_id = netbox_cooling_outflow.out2.id
  description        = "${local.prefix} second coolant supply"
}

resource "netbox_interface" "eth0" {
  device_id   = netbox_device.sw01.id
  name        = "eth0"
  type        = "1000base-t"
  description = "${local.prefix} management interface"
}

resource "netbox_mac_address" "eth0" {
  mac_address          = "02:00:5E:DC:1A:01"
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.eth0.id
  description          = "${local.prefix} MAC of eth0"
}

resource "netbox_cable_bundle" "a01_trunk" {
  name        = "${local.prefix} A01 trunk"
  description = "${local.prefix} trunk from rack A01 to the MDA"
}

# --- Read back through data sources -----------------------------------------

data "netbox_site" "fra1" {
  slug = netbox_site.fra1.slug
}

data "netbox_power_feeds" "a01" {
  filters    = [{ name = "rack_id", value = tostring(netbox_rack.a01.id) }]
  depends_on = [netbox_power_feed.a01_a, netbox_power_feed.a01_b]
}

output "site" {
  value = {
    id     = data.netbox_site.fra1.id
    name   = data.netbox_site.fra1.name
    region = netbox_region.germany.name
  }
}

output "rack_a01" {
  value = {
    id       = netbox_rack.a01.id
    u_height = netbox_rack.a01.u_height # inherited from the rack type
    width    = netbox_rack.a01.width
    feeds    = data.netbox_power_feeds.a01.items[*].name
  }
}

output "device" {
  value = {
    name = netbox_device.sw01.name
    mac  = netbox_mac_address.eth0.mac_address
  }
}
