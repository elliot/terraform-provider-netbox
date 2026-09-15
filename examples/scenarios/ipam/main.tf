# End-to-end IPAM scenario: a small data centre address plan.
#
#   RIR ─ aggregate ─ container prefix ─ child prefixes ─ IP range / addresses
#   VRF with import/export route targets
#   VLAN group ─ VLANs (bound to prefixes)
#   ASN range ─ ASN
#   FHRP (VRRP) group assigned to a router interface, service on the router
#
# Everything is prefixed "tfacc-ipam-scenario" so it is easy to find and
# sweep on a shared instance. Apply and destroy with:
#
#   export NETBOX_SERVER_URL=https://demo.netbox.dev NETBOX_API_TOKEN=...
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
  prefix = "tfacc-ipam-scenario"
}

# --- Tenancy -----------------------------------------------------------------

resource "netbox_tenant" "acme" {
  name = "${local.prefix} ACME"
  slug = "${local.prefix}-acme"
}

# --- Registries and aggregates ----------------------------------------------

resource "netbox_rir" "private" {
  name        = "${local.prefix} RFC 1918"
  slug        = "${local.prefix}-rfc1918"
  is_private  = true
  description = "Private address space (scenario)"
}

# Aggregates may not overlap; 2001:db8:5ce::/48 is documentation space that
# no other demo object uses.
resource "netbox_aggregate" "v6" {
  prefix      = "2001:db8:5ce::/48"
  rir_id      = netbox_rir.private.id
  tenant_id   = netbox_tenant.acme.id
  date_added  = "2024-01-15"
  description = "${local.prefix} IPv6 documentation aggregate"
}

# --- VRF with route targets --------------------------------------------------

resource "netbox_route_target" "import" {
  name        = "4200000213:1"
  tenant_id   = netbox_tenant.acme.id
  description = "${local.prefix} import"
}

resource "netbox_route_target" "export" {
  name        = "4200000213:2"
  tenant_id   = netbox_tenant.acme.id
  description = "${local.prefix} export"
}

resource "netbox_vrf" "acme" {
  name              = "${local.prefix}-acme-l3vpn"
  rd                = "4200000213:213"
  tenant_id         = netbox_tenant.acme.id
  enforce_unique    = true
  import_target_ids = [netbox_route_target.import.id]
  export_target_ids = [netbox_route_target.export.id]
  description       = "${local.prefix} customer VRF"
}

# --- Roles -------------------------------------------------------------------

resource "netbox_ipam_role" "servers" {
  name   = "${local.prefix} Servers"
  slug   = "${local.prefix}-servers"
  weight = 1000
}

resource "netbox_ipam_role" "dhcp" {
  name   = "${local.prefix} DHCP"
  slug   = "${local.prefix}-dhcp"
  weight = 500
}

# --- Site, device and interface (for VLAN scope, FHRP and service) -----------

resource "netbox_site" "dc1" {
  name      = "${local.prefix} DC1"
  slug      = "${local.prefix}-dc1"
  tenant_id = netbox_tenant.acme.id

  # netbox_asn.core1 attaches itself to this site through site_ids; the same
  # relation is exposed here as asn_ids, so let the ASN own it.
  lifecycle {
    ignore_changes = [asn_ids]
  }
}

resource "netbox_manufacturer" "generic" {
  name = "${local.prefix} Generic"
  slug = "${local.prefix}-generic"
}

resource "netbox_device_type" "router" {
  manufacturer_id = netbox_manufacturer.generic.id
  model           = "${local.prefix} Router"
  slug            = "${local.prefix}-router"
}

resource "netbox_device_role" "router" {
  name = "${local.prefix} Router"
  slug = "${local.prefix}-router"
}

resource "netbox_device" "core1" {
  name           = "${local.prefix}-core1"
  device_type_id = netbox_device_type.router.id
  role_id        = netbox_device_role.router.id
  site_id        = netbox_site.dc1.id
  tenant_id      = netbox_tenant.acme.id
  status         = "active"
}

resource "netbox_interface" "core1_vlan20" {
  device_id   = netbox_device.core1.id
  name        = "vlan20"
  type        = "virtual"
  description = "${local.prefix} SVI for servers VLAN"
}

# --- VLAN group and VLANs ----------------------------------------------------

resource "netbox_vlan_group" "dc1" {
  name       = "${local.prefix} DC1 VLANs"
  slug       = "${local.prefix}-dc1-vlans"
  scope_type = "dcim.site"
  scope_id   = netbox_site.dc1.id
  vid_ranges = [[20, 29], [100, 199]]
}

resource "netbox_vlan" "servers" {
  group_id  = netbox_vlan_group.dc1.id
  vid       = 20
  name      = "${local.prefix}-servers"
  tenant_id = netbox_tenant.acme.id
  role_id   = netbox_ipam_role.servers.id
  status    = "active"
}

resource "netbox_vlan" "mgmt" {
  group_id = netbox_vlan_group.dc1.id
  vid      = 100
  name     = "${local.prefix}-mgmt"
  status   = "reserved"
}

# --- Prefix hierarchy --------------------------------------------------------

resource "netbox_prefix" "supernet" {
  prefix      = "10.213.0.0/16"
  vrf_id      = netbox_vrf.acme.id
  tenant_id   = netbox_tenant.acme.id
  scope_type  = "dcim.site"
  scope_id    = netbox_site.dc1.id
  status      = "container"
  description = "${local.prefix} DC1 supernet"
}

resource "netbox_prefix" "servers" {
  prefix      = "10.213.20.0/24"
  vrf_id      = netbox_vrf.acme.id
  tenant_id   = netbox_tenant.acme.id
  scope_type  = "dcim.site"
  scope_id    = netbox_site.dc1.id
  vlan_id     = netbox_vlan.servers.id
  role_id     = netbox_ipam_role.servers.id
  status      = "active"
  description = "${local.prefix} servers"
  depends_on  = [netbox_prefix.supernet]
}

resource "netbox_prefix" "loopbacks" {
  prefix      = "10.213.255.0/24"
  vrf_id      = netbox_vrf.acme.id
  tenant_id   = netbox_tenant.acme.id
  status      = "active"
  is_pool     = true
  description = "${local.prefix} loopbacks"
  depends_on  = [netbox_prefix.supernet]
}

# --- IP range and addresses --------------------------------------------------

resource "netbox_ip_range" "dhcp" {
  start_address = "10.213.20.100/24"
  end_address   = "10.213.20.199/24"
  vrf_id        = netbox_vrf.acme.id
  tenant_id     = netbox_tenant.acme.id
  role_id       = netbox_ipam_role.dhcp.id
  status        = "active"
  mark_utilized = true
  description   = "${local.prefix} DHCP pool"
  depends_on    = [netbox_prefix.servers]
}

resource "netbox_ip_address" "gateway_vip" {
  address     = "10.213.20.1/24"
  vrf_id      = netbox_vrf.acme.id
  tenant_id   = netbox_tenant.acme.id
  status      = "active"
  role        = "vip"
  dns_name    = "gw.servers.dc1.example.com"
  description = "${local.prefix} VRRP VIP"
  depends_on  = [netbox_prefix.servers]
}

resource "netbox_ip_address" "core1_vlan20" {
  address              = "10.213.20.2/24"
  vrf_id               = netbox_vrf.acme.id
  tenant_id            = netbox_tenant.acme.id
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.core1_vlan20.id
  dns_name             = "core1-vlan20.dc1.example.com"
  description          = "${local.prefix} core1 SVI"
  depends_on           = [netbox_prefix.servers]
}

resource "netbox_ip_address" "core1_loopback" {
  address     = "10.213.255.1/32"
  vrf_id      = netbox_vrf.acme.id
  tenant_id   = netbox_tenant.acme.id
  role        = "loopback"
  dns_name    = "core1.dc1.example.com"
  description = "${local.prefix} core1 loopback"
  depends_on  = [netbox_prefix.loopbacks]
}

# --- ASN range and ASN -------------------------------------------------------

resource "netbox_asn_range" "fabric" {
  name        = "${local.prefix} fabric"
  slug        = "${local.prefix}-fabric"
  rir_id      = netbox_rir.private.id
  tenant_id   = netbox_tenant.acme.id
  start       = 4200213000
  end         = 4200213099
  description = "${local.prefix} private 4-byte ASNs"
}

resource "netbox_asn" "core1" {
  asn         = 4200213001
  rir_id      = netbox_rir.private.id
  tenant_id   = netbox_tenant.acme.id
  site_ids    = [netbox_site.dc1.id]
  description = "${local.prefix} core1"
  depends_on  = [netbox_asn_range.fabric]
}

# --- FHRP group on the SVI ---------------------------------------------------

resource "netbox_fhrp_group" "servers_gw" {
  name        = "${local.prefix}-servers-gw"
  protocol    = "vrrp3"
  group_id    = 20
  auth_type   = "plaintext"
  auth_key    = "scenario"
  description = "${local.prefix} servers gateway"
}

resource "netbox_fhrp_group_assignment" "core1" {
  group_id       = netbox_fhrp_group.servers_gw.id
  interface_type = "dcim.interface"
  interface_id   = netbox_interface.core1_vlan20.id
  priority       = 200
}

# --- Service on the router ---------------------------------------------------

resource "netbox_service_template" "ssh" {
  name          = "${local.prefix} SSH"
  port_mappings = ["tcp/22"]
}

resource "netbox_service" "core1_ssh" {
  parent_object_type = "dcim.device"
  parent_object_id   = netbox_device.core1.id
  name               = "${local.prefix}-ssh"
  port_mappings      = netbox_service_template.ssh.port_mappings
  ipaddress_ids      = [netbox_ip_address.core1_vlan20.id]
  description        = "${local.prefix} management SSH"
}

# --- Read back ---------------------------------------------------------------

data "netbox_prefixes" "under_supernet" {
  filters = [
    { name = "within", value = netbox_prefix.supernet.prefix },
    { name = "vrf_id", value = tostring(netbox_vrf.acme.id) },
  ]
  depends_on = [netbox_prefix.servers, netbox_prefix.loopbacks]
}

data "netbox_ip_addresses" "servers_vlan" {
  filters = [
    { name = "parent", value = netbox_prefix.servers.prefix },
    { name = "vrf_id", value = tostring(netbox_vrf.acme.id) },
  ]
  depends_on = [netbox_ip_address.gateway_vip, netbox_ip_address.core1_vlan20]
}

output "child_prefixes" {
  value = data.netbox_prefixes.under_supernet.items[*].prefix
}

output "servers_vlan_addresses" {
  value = data.netbox_ip_addresses.servers_vlan.items[*].address
}

output "vrf_id" {
  value = netbox_vrf.acme.id
}
