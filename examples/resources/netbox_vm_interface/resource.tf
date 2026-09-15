resource "netbox_cluster_type" "vmware" {
  name = "VMware vSphere"
  slug = "vmware-vsphere"
}

resource "netbox_cluster" "fra1_prod" {
  name    = "fra1-prod-01"
  type_id = netbox_cluster_type.vmware.id
}

resource "netbox_virtual_machine" "web01" {
  name       = "web01.example.com"
  cluster_id = netbox_cluster.fra1_prod.id
}

resource "netbox_vlan" "servers" {
  name = "servers"
  vid  = 100
}

resource "netbox_vlan" "backup" {
  name = "backup"
  vid  = 200
}

resource "netbox_vm_interface" "eth0" {
  virtual_machine_id = netbox_virtual_machine.web01.id
  name               = "eth0"
  mtu                = 1500
  mode               = "tagged"
  untagged_vlan_id   = netbox_vlan.servers.id
  tagged_vlan_ids    = [netbox_vlan.backup.id]
  description        = "Primary NIC"
}

# A child interface (VLAN sub-interface) on eth0.
resource "netbox_vm_interface" "eth0_200" {
  virtual_machine_id = netbox_virtual_machine.web01.id
  name               = "eth0.200"
  parent_id          = netbox_vm_interface.eth0.id
  mode               = "access"
  untagged_vlan_id   = netbox_vlan.backup.id
}

# Assign an address to the interface and use it as the VM's primary IPv4.
resource "netbox_ip_address" "web01" {
  address              = "10.0.100.10/24"
  assigned_object_type = "virtualization.vminterface"
  assigned_object_id   = netbox_vm_interface.eth0.id
  dns_name             = "web01.example.com"
}
