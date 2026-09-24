resource "netbox_route_target" "tenant_a" {
  name = "65000:100"
}

# VXLAN EVPN overlay with the VNI as identifier.
resource "netbox_l2vpn" "tenant_a" {
  name              = "tenant-a-overlay"
  slug              = "tenant-a-overlay"
  description       = "Layer 2 overlay for tenant A"
  type              = "vxlan-evpn" # vxlan, vpls, mpls-evpn, evpn-vpws, ...
  identifier        = 100100
  status            = "active"
  import_target_ids = [netbox_route_target.tenant_a.id]
  export_target_ids = [netbox_route_target.tenant_a.id]
  tenant_id         = netbox_tenant.tenant_a.id
}
