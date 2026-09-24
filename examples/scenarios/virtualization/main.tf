# Scenario: a small virtualization estate plus the ownership and access
# objects around it. Validated with `terraform apply` / `terraform destroy`
# against https://demo.netbox.dev (NetBox 4.7.0).
#
# Objects: cluster type + group, a site-scoped cluster, a VM type, a VM with
# two interfaces and two virtual disks, an IP address assigned to the primary
# interface, an owner group + owner (NetBox 4.7 ownership), and a user group +
# user granted a scoped object permission.

terraform {
  required_providers {
    netbox = {
      source = "elliot/netbox"
    }
  }
}

# Configured from NETBOX_SERVER_URL / NETBOX_API_TOKEN (and optionally
# NETBOX_REQUESTS_PER_SECOND).
provider "netbox" {}

variable "prefix" {
  description = "Prefix for every object name/slug so the scenario is easy to find and clean up."
  type        = string
  default     = "tfacc-scn"
}

variable "user_password" {
  description = "Initial password of the scenario user."
  type        = string
  sensitive   = true
  default     = "ChangeMe-Now-2026!"
}

locals {
  p = var.prefix
}

# ----------------------------------------------------------------------------
# Ownership (NetBox 4.7): an owner is a team that can be set as owner_id on
# most objects.
# ----------------------------------------------------------------------------

resource "netbox_owner_group" "infra" {
  name        = "${local.p} Infrastructure"
  description = "${local.p}: teams operating the virtual infrastructure"
}

resource "netbox_owner" "virt_team" {
  name           = "${local.p} Virtualization team"
  group_id       = netbox_owner_group.infra.id
  description    = "${local.p}: owns hypervisors and VMs"
  user_group_ids = [netbox_user_group.virt_ops.id]
  user_ids       = [netbox_user.operator.id]
}

# ----------------------------------------------------------------------------
# Access: a group, a user in it, and a permission granted to the group.
# ----------------------------------------------------------------------------

resource "netbox_user_group" "virt_ops" {
  name        = "${local.p}-virt-ops"
  description = "${local.p}: virtualization operators"
}

resource "netbox_user" "operator" {
  username   = "${local.p}-operator"
  password   = var.user_password
  first_name = "Scenario"
  last_name  = "Operator"
  email      = "${local.p}-operator@example.com"
  group_ids  = [netbox_user_group.virt_ops.id]
}

resource "netbox_permission" "virt_ops_manage_vms" {
  name         = "${local.p}: manage virtual machines"
  description  = "${local.p}: full control over VMs, interfaces and disks"
  object_types = ["virtualization.virtualmachine", "virtualization.vminterface", "virtualization.virtualdisk"]
  actions      = ["view", "add", "change", "delete"]
  constraints  = jsonencode({ cluster__group__slug = "${local.p}-emea" })
  group_ids    = [netbox_user_group.virt_ops.id]
}

# ----------------------------------------------------------------------------
# Virtualization
# ----------------------------------------------------------------------------

resource "netbox_site" "fra1" {
  name        = "${local.p}-fra1"
  slug        = "${local.p}-fra1"
  status      = "active"
  description = "${local.p}: Frankfurt"
}

resource "netbox_cluster_type" "vmware" {
  name        = "${local.p} VMware vSphere"
  slug        = "${local.p}-vmware-vsphere"
  description = "${local.p}: vSphere 8"
}

resource "netbox_cluster_group" "emea" {
  name        = "${local.p} EMEA"
  slug        = "${local.p}-emea"
  description = "${local.p}: European clusters"
}

resource "netbox_cluster" "fra1_prod" {
  name        = "${local.p}-fra1-prod-01"
  type_id     = netbox_cluster_type.vmware.id
  group_id    = netbox_cluster_group.emea.id
  status      = "active"
  scope_type  = "dcim.site"
  scope_id    = netbox_site.fra1.id
  owner_id    = netbox_owner.virt_team.id
  description = "${local.p}: production cluster"
}

resource "netbox_virtual_machine_type" "m_large" {
  name        = "${local.p} m.large"
  slug        = "${local.p}-m-large"
  description = "${local.p}: 2 vCPU / 8 GB"
}

resource "netbox_virtual_machine" "web01" {
  name                    = "${local.p}-web01"
  cluster_id              = netbox_cluster.fra1_prod.id
  site_id                 = netbox_site.fra1.id
  virtual_machine_type_id = netbox_virtual_machine_type.m_large.id
  owner_id                = netbox_owner.virt_team.id
  status                  = "active"
  start_on_boot           = "on"
  vcpus                   = 2
  memory                  = 8192
  description             = "${local.p}: front-end web server"

  local_context_data = jsonencode({
    ntp_servers = ["10.0.0.1", "10.0.0.2"]
  })
}

resource "netbox_vm_interface" "eth0" {
  virtual_machine_id = netbox_virtual_machine.web01.id
  name               = "eth0"
  mtu                = 1500
  description        = "${local.p}: primary NIC"
}

resource "netbox_vm_interface" "eth1" {
  virtual_machine_id = netbox_virtual_machine.web01.id
  name               = "eth1"
  enabled            = false
  description        = "${local.p}: backup network (disabled)"
}

resource "netbox_virtual_disk" "root" {
  virtual_machine_id = netbox_virtual_machine.web01.id
  name               = "root"
  size               = 40960
  description        = "${local.p}: OS disk"
}

resource "netbox_virtual_disk" "data" {
  virtual_machine_id = netbox_virtual_machine.web01.id
  name               = "data"
  size               = 204800
  description        = "${local.p}: application data"
}

resource "netbox_ip_address" "web01_eth0" {
  address              = "10.123.45.10/24"
  status               = "active"
  assigned_object_type = "virtualization.vminterface"
  assigned_object_id   = netbox_vm_interface.eth0.id
  dns_name             = "${local.p}-web01.example.com"
  description          = "${local.p}: web01 eth0"
}

# ----------------------------------------------------------------------------
# Outputs
# ----------------------------------------------------------------------------

output "virtual_machine_id" {
  value = netbox_virtual_machine.web01.id
}

output "virtual_machine_disk_mb" {
  description = "Computed by NetBox from the attached virtual disks."
  value       = netbox_virtual_machine.web01.disk
}

output "web01_address" {
  value = netbox_ip_address.web01_eth0.address
}

output "operator_user_id" {
  value = netbox_user.operator.id
}
