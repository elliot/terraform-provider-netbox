resource "netbox_cluster_type" "vmware" {
  name = "VMware vSphere"
  slug = "vmware-vsphere"
}

resource "netbox_cluster" "fra1_prod" {
  name    = "fra1-prod-01"
  type_id = netbox_cluster_type.vmware.id
}

resource "netbox_device_role" "app_server" {
  name = "Application server"
  slug = "application-server"
}

resource "netbox_platform" "ubuntu" {
  name = "Ubuntu 24.04"
  slug = "ubuntu-24-04"
}

resource "netbox_virtual_machine_type" "m_large" {
  name = "m.large"
  slug = "m-large"
}

# A virtual machine needs a cluster and/or a site. vcpus accepts decimals,
# memory and disk are in MB. Once netbox_virtual_disk resources are attached
# NetBox computes disk from them, so leave it unset in that case.
resource "netbox_virtual_machine" "web01" {
  name                    = "web01.example.com"
  cluster_id              = netbox_cluster.fra1_prod.id
  role_id                 = netbox_device_role.app_server.id
  platform_id             = netbox_platform.ubuntu.id
  virtual_machine_type_id = netbox_virtual_machine_type.m_large.id
  status                  = "active"
  start_on_boot           = "on"
  vcpus                   = 2
  memory                  = 8192
  description             = "Front-end web server"

  local_context_data = jsonencode({
    ntp_servers = ["10.0.0.1", "10.0.0.2"]
  })

  tags = ["web"]
}
