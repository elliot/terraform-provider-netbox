resource "netbox_vrf" "acme" {
  name = "ACME-L3VPN"
}

# A standalone address (e.g. a VIP) ...
resource "netbox_ip_address" "vip" {
  address     = "10.10.20.1/24"
  vrf_id      = netbox_vrf.acme.id
  status      = "active"
  role        = "vip"
  dns_name    = "gw.office.example.com"
  description = "Office gateway VIP"
}

# ... and one assigned to a device interface.
resource "netbox_ip_address" "loopback" {
  address              = "10.10.255.1/32"
  vrf_id               = netbox_vrf.acme.id
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.lo0.id
  role                 = "loopback"
  dns_name             = "leaf1.example.com"
}
