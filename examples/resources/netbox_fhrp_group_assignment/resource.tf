resource "netbox_fhrp_group" "office_gw" {
  name     = "office-gateway"
  protocol = "vrrp3"
  group_id = 20
}

# Assign the group to a device interface (dcim.interface) or a VM interface
# (virtualization.vminterface). Lower priority = backup router.
resource "netbox_fhrp_group_assignment" "primary" {
  group_id       = netbox_fhrp_group.office_gw.id
  interface_type = "dcim.interface"
  interface_id   = netbox_interface.core1_vlan20.id
  priority       = 200
}

resource "netbox_fhrp_group_assignment" "backup" {
  group_id       = netbox_fhrp_group.office_gw.id
  interface_type = "dcim.interface"
  interface_id   = netbox_interface.core2_vlan20.id
  priority       = 100
}
