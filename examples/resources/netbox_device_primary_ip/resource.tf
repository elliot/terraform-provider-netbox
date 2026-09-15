# Hand-maintained example for netbox_device_primary_ip.

resource "netbox_manufacturer" "arista" {
  name = "Arista"
  slug = "arista"
}

resource "netbox_device_type" "dcs_7050" {
  manufacturer_id = netbox_manufacturer.arista.id
  model           = "DCS-7050SX3-48YC8"
  slug            = "dcs-7050sx3-48yc8"
}

resource "netbox_device_role" "leaf" {
  name = "Leaf"
  slug = "leaf"
}

resource "netbox_site" "dc1" {
  name = "DC1"
  slug = "dc1"
}

resource "netbox_device" "leaf1" {
  name           = "leaf1"
  device_type_id = netbox_device_type.dcs_7050.id
  role_id        = netbox_device_role.leaf.id
  site_id        = netbox_site.dc1.id
  # Do not set primary_ip4_id / primary_ip6_id here: the address depends on the
  # interface, which depends on this device.
}

resource "netbox_interface" "leaf1_mgmt" {
  device_id = netbox_device.leaf1.id
  name      = "Management1"
  type      = "1000base-t"
  mgmt_only = true
}

resource "netbox_ip_address" "leaf1_mgmt" {
  address              = "10.0.0.11/24"
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.leaf1_mgmt.id
  dns_name             = "leaf1.example.com"
}

# The IP version (4) is detected from the address.
resource "netbox_device_primary_ip" "leaf1" {
  device_id     = netbox_device.leaf1.id
  ip_address_id = netbox_ip_address.leaf1_mgmt.id
}
