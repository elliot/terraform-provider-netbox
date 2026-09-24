resource "netbox_manufacturer" "example" {
  name = "Example Networks"
  slug = "example-networks"
}

resource "netbox_device_type" "sw48" {
  manufacturer_id = netbox_manufacturer.example.id
  model           = "SW-48"
  slug            = "example-sw-48"
  u_height        = 1
}

resource "netbox_device_role" "access_switch" {
  name  = "Access switch"
  slug  = "access-switch"
  color = "2196f3"
}

resource "netbox_site" "fra1" {
  name = "FRA1"
  slug = "fra1"
}

resource "netbox_device" "sw01" {
  name           = "fra1-sw01"
  device_type_id = netbox_device_type.sw48.id
  role_id        = netbox_device_role.access_switch.id
  site_id        = netbox_site.fra1.id
}

resource "netbox_interface" "eth0" {
  device_id = netbox_device.sw01.id
  name      = "eth0"
  type      = "1000base-t"
}

# A MAC address assigned to a device interface. Use
# assigned_object_type = "virtualization.vminterface" for VM interfaces.
resource "netbox_mac_address" "eth0" {
  mac_address          = "00:1B:44:11:3A:B7"
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.eth0.id
  description          = "Burned-in address of fra1-sw01 eth0"
}
