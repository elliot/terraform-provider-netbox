# Hand-maintained example for netbox_virtual_machine_primary_ip.

resource "netbox_cluster_type" "vmware" {
  name = "VMware vSphere"
  slug = "vmware-vsphere"
}

resource "netbox_cluster" "dc1" {
  name    = "dc1-cluster"
  type_id = netbox_cluster_type.vmware.id
}

resource "netbox_virtual_machine" "app01" {
  name       = "app01"
  cluster_id = netbox_cluster.dc1.id
  vcpus      = 4
  memory     = 8192
  # Do not set primary_ip4_id / primary_ip6_id here: the address depends on the
  # interface, which depends on this virtual machine.
}

resource "netbox_vm_interface" "app01_eth0" {
  virtual_machine_id = netbox_virtual_machine.app01.id
  name               = "eth0"
}

resource "netbox_ip_address" "app01_v6" {
  address              = "2001:db8:10::21/64"
  assigned_object_type = "virtualization.vminterface"
  assigned_object_id   = netbox_vm_interface.app01_eth0.id
}

resource "netbox_virtual_machine_primary_ip" "app01" {
  virtual_machine_id = netbox_virtual_machine.app01.id
  ip_address_id      = netbox_ip_address.app01_v6.id
  ip_address_version = 6 # optional; detected from the address when omitted
}
