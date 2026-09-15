# Cluster types describe the virtualization technology of a cluster.
resource "netbox_cluster_type" "vmware" {
  name        = "VMware vSphere"
  slug        = "vmware-vsphere"
  description = "vSphere 8 clusters"
}

resource "netbox_cluster_type" "proxmox" {
  name = "Proxmox VE"
  slug = "proxmox-ve"
}
