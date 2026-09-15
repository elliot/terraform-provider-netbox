resource "netbox_virtual_circuit_type" "example" {
  name        = "L2VPN"
  slug        = "l2vpn"
  color       = "9c27b0"
  description = "Point-to-point Ethernet pseudowires"
}
