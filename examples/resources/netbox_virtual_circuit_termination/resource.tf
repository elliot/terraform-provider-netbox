resource "netbox_provider" "example" {
  name = "Lumen"
  slug = "lumen"
}

resource "netbox_provider_network" "example" {
  provider_id = netbox_provider.example.id
  name        = "Lumen MPLS EU"
}

resource "netbox_virtual_circuit_type" "example" {
  name = "L2VPN"
  slug = "l2vpn"
}

resource "netbox_virtual_circuit" "example" {
  cid                 = "LUM-L2VPN-9001"
  provider_network_id = netbox_provider_network.example.id
  type_id             = netbox_virtual_circuit_type.example.id
}

# Virtual circuits terminate on virtual interfaces of a device; the device
# chain (manufacturer, device type, role, site) is omitted here for brevity.
resource "netbox_interface" "ams_pw" {
  device_id = netbox_device.ams_edge.id
  name      = "pw9001"
  type      = "virtual"
}

resource "netbox_interface" "fra_pw" {
  device_id = netbox_device.fra_edge.id
  name      = "pw9001"
  type      = "virtual"
}

resource "netbox_virtual_circuit_termination" "ams" {
  virtual_circuit_id = netbox_virtual_circuit.example.id
  interface_id       = netbox_interface.ams_pw.id
  role               = "peer"
}

resource "netbox_virtual_circuit_termination" "fra" {
  virtual_circuit_id = netbox_virtual_circuit.example.id
  interface_id       = netbox_interface.fra_pw.id
  role               = "peer"
}
