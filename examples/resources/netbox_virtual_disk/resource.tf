resource "netbox_cluster_type" "vmware" {
  name = "VMware vSphere"
  slug = "vmware-vsphere"
}

resource "netbox_cluster" "fra1_prod" {
  name    = "fra1-prod-01"
  type_id = netbox_cluster_type.vmware.id
}

resource "netbox_virtual_machine" "db01" {
  name       = "db01.example.com"
  cluster_id = netbox_cluster.fra1_prod.id
}

# Sizes are in MB. NetBox sets the VM's disk attribute to the sum of its
# virtual disks.
resource "netbox_virtual_disk" "root" {
  virtual_machine_id = netbox_virtual_machine.db01.id
  name               = "root"
  size               = 40960
  description        = "OS disk"
}

resource "netbox_virtual_disk" "data" {
  virtual_machine_id = netbox_virtual_machine.db01.id
  name               = "data"
  size               = 512000
  description        = "PostgreSQL data volume"
}
