# Circuits and tenancy scenario
#
# Models a customer (tenant) whose Amsterdam office is connected by a
# provider-delivered internet circuit terminating at the site (A side) and in
# the provider's MPLS cloud (Z side), grouped with other uplinks, plus an
# L2VPN virtual circuit riding on that provider network and terminating on
# virtual interfaces of the site's edge router. A technical contact is
# attached to the site.
#
# Validated against https://demo.netbox.dev with the locally built provider:
#   export NETBOX_SERVER_URL=https://demo.netbox.dev NETBOX_API_TOKEN=...
#   terraform init && terraform apply && terraform destroy

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
  default     = "tfacc-circuits"
}

# ------------------------------------------------------------------ tenancy

resource "netbox_tenant_group" "customers" {
  name        = "${var.prefix} Customers"
  slug        = "${var.prefix}-customers"
  description = "External customers"
}

resource "netbox_tenant" "acme" {
  name        = "${var.prefix} Acme Corp"
  slug        = "${var.prefix}-acme"
  group_id    = netbox_tenant_group.customers.id
  description = "Managed services customer"
}

resource "netbox_contact_group" "acme_noc" {
  name = "${var.prefix} Acme NOC"
  slug = "${var.prefix}-acme-noc"
}

resource "netbox_contact_role" "technical" {
  name = "${var.prefix} Technical"
  slug = "${var.prefix}-technical"
}

resource "netbox_contact" "jane" {
  name      = "${var.prefix} Jane Doe"
  group_ids = [netbox_contact_group.acme_noc.id]
  title     = "Network Operations Manager"
  email     = "jane.doe@acme.example"
  phone     = "+31 20 555 0100"
}

# --------------------------------------------------------------------- site

resource "netbox_site" "ams" {
  name      = "${var.prefix} Amsterdam office"
  slug      = "${var.prefix}-ams"
  tenant_id = netbox_tenant.acme.id
  status    = "active"
}

resource "netbox_contact_assignment" "ams_technical" {
  object_type = "dcim.site"
  object_id   = netbox_site.ams.id
  contact_id  = netbox_contact.jane.id
  role_id     = netbox_contact_role.technical.id
  priority    = "primary"
}

# ----------------------------------------------------------------- provider

resource "netbox_provider" "lumen" {
  name        = "${var.prefix} Lumen"
  slug        = "${var.prefix}-lumen"
  description = "Transport and internet access provider"
}

resource "netbox_provider_account" "lumen_emea" {
  provider_id = netbox_provider.lumen.id
  account     = "${var.prefix}-ACCT-00812345"
  name        = "${var.prefix} EMEA billing"
}

resource "netbox_provider_network" "lumen_mpls" {
  provider_id = netbox_provider.lumen.id
  name        = "${var.prefix} Lumen MPLS EU"
  service_id  = "MPLS-EU-01"
}

# ----------------------------------------------------------------- circuits

resource "netbox_circuit_type" "dia" {
  name  = "${var.prefix} Internet access"
  slug  = "${var.prefix}-internet-access"
  color = "2196f3"
}

resource "netbox_circuit" "ams_dia" {
  cid                 = "${var.prefix}-LUM-DIA-48213"
  provider_id         = netbox_provider.lumen.id
  provider_account_id = netbox_provider_account.lumen_emea.id
  type_id             = netbox_circuit_type.dia.id
  tenant_id           = netbox_tenant.acme.id
  status              = "active"
  install_date        = "2024-03-01"
  commit_rate         = 1000000
  description         = "1G DIA for the Amsterdam office"
}

resource "netbox_circuit_termination" "ams_dia_a" {
  circuit_id       = netbox_circuit.ams_dia.id
  term_side        = "A"
  termination_type = "dcim.site"
  termination_id   = netbox_site.ams.id
  port_speed       = 1000000
  xconnect_id      = "XC-AMS-0142"
  pp_info          = "MMR PP03 ports 7-8"
}

resource "netbox_circuit_termination" "ams_dia_z" {
  circuit_id       = netbox_circuit.ams_dia.id
  term_side        = "Z"
  termination_type = "circuits.providernetwork"
  termination_id   = netbox_provider_network.lumen_mpls.id
}

resource "netbox_circuit_group" "ams_uplinks" {
  name      = "${var.prefix} AMS uplinks"
  slug      = "${var.prefix}-ams-uplinks"
  tenant_id = netbox_tenant.acme.id
}

resource "netbox_circuit_group_assignment" "ams_dia" {
  group_id    = netbox_circuit_group.ams_uplinks.id
  member_type = "circuits.circuit"
  member_id   = netbox_circuit.ams_dia.id
  priority    = "primary"
}

# --------------------------------------------------------- virtual circuits

resource "netbox_manufacturer" "juniper" {
  name = "${var.prefix} Juniper"
  slug = "${var.prefix}-juniper"
}

resource "netbox_device_type" "mx204" {
  manufacturer_id = netbox_manufacturer.juniper.id
  model           = "${var.prefix} MX204"
  slug            = "${var.prefix}-mx204"
}

resource "netbox_device_role" "edge" {
  name = "${var.prefix} Edge router"
  slug = "${var.prefix}-edge-router"
}

resource "netbox_device" "ams_edge" {
  name           = "${var.prefix}-ams-edge-01"
  device_type_id = netbox_device_type.mx204.id
  role_id        = netbox_device_role.edge.id
  site_id        = netbox_site.ams.id
  tenant_id      = netbox_tenant.acme.id
}

resource "netbox_interface" "pw_hub" {
  device_id   = netbox_device.ams_edge.id
  name        = "pw9001.hub"
  type        = "virtual"
  description = "${var.prefix} L2VPN hub"
}

resource "netbox_interface" "pw_spoke" {
  device_id   = netbox_device.ams_edge.id
  name        = "pw9001.spoke"
  type        = "virtual"
  description = "${var.prefix} L2VPN spoke"
}

resource "netbox_virtual_circuit_type" "l2vpn" {
  name = "${var.prefix} L2VPN"
  slug = "${var.prefix}-l2vpn"
}

resource "netbox_virtual_circuit" "pw9001" {
  cid                 = "${var.prefix}-LUM-L2VPN-9001"
  provider_network_id = netbox_provider_network.lumen_mpls.id
  provider_account_id = netbox_provider_account.lumen_emea.id
  type_id             = netbox_virtual_circuit_type.l2vpn.id
  tenant_id           = netbox_tenant.acme.id
  status              = "active"
  description         = "Ethernet pseudowire over the Lumen MPLS cloud"
}

resource "netbox_virtual_circuit_termination" "hub" {
  virtual_circuit_id = netbox_virtual_circuit.pw9001.id
  interface_id       = netbox_interface.pw_hub.id
  role               = "hub"
}

resource "netbox_virtual_circuit_termination" "spoke" {
  virtual_circuit_id = netbox_virtual_circuit.pw9001.id
  interface_id       = netbox_interface.pw_spoke.id
  role               = "spoke"
}

# ------------------------------------------------------------ data sources

data "netbox_circuits" "acme" {
  filters = [
    { name = "tenant_id", value = tostring(netbox_tenant.acme.id) },
  ]
  depends_on = [netbox_circuit.ams_dia]
}

data "netbox_contact_assignments" "ams" {
  filters = [
    { name = "object_type", value = "dcim.site" },
    { name = "object_id", value = tostring(netbox_site.ams.id) },
  ]
  depends_on = [netbox_contact_assignment.ams_technical]
}

output "acme_circuit_cids" {
  value = data.netbox_circuits.acme.items[*].cid
}

output "ams_contact_ids" {
  value = data.netbox_contact_assignments.ams.items[*].contact_id
}

output "ams_dia_terminations" {
  value = {
    a = netbox_circuit_termination.ams_dia_a.id
    z = netbox_circuit_termination.ams_dia_z.id
  }
}
