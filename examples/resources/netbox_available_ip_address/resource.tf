# Hand-maintained example for netbox_available_ip_address.

resource "netbox_prefix" "servers" {
  prefix      = "10.20.30.0/24"
  status      = "active"
  description = "Server LAN"
}

# Allocate the next free address of the prefix and assign it to an interface.
resource "netbox_available_ip_address" "web01" {
  prefix_id            = netbox_prefix.servers.id
  dns_name             = "web01.example.com"
  description          = "web01 management"
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.web01_eth0.id
  tags                 = ["managed-by-terraform"]
}

# Allocation from an IP range works the same way.
resource "netbox_ip_range" "dhcp_static" {
  start_address = "10.20.30.200/24"
  end_address   = "10.20.30.250/24"
}

resource "netbox_available_ip_address" "printer" {
  ip_range_id = netbox_ip_range.dhcp_static.id
  status      = "reserved"
  description = "Lobby printer"
}

output "web01_address" {
  value = netbox_available_ip_address.web01.address # e.g. "10.20.30.1/24"
}
