resource "netbox_cluster_type" "vmware" {
  name = "VMware vSphere"
  slug = "vmware-vsphere"
}

resource "netbox_cluster_group" "emea" {
  name = "EMEA"
  slug = "emea"
}

resource "netbox_site" "fra1" {
  name = "FRA1"
  slug = "fra1"
}

resource "netbox_tenant" "platform" {
  name = "Platform Engineering"
  slug = "platform-engineering"
}

# A cluster scoped to a site: virtual machines in this cluster inherit the
# site. scope_type/scope_id are optional (dcim.region, dcim.sitegroup,
# dcim.site and dcim.location are accepted).
resource "netbox_cluster" "fra1_prod" {
  name       = "fra1-prod-01"
  type_id    = netbox_cluster_type.vmware.id
  group_id   = netbox_cluster_group.emea.id
  tenant_id  = netbox_tenant.platform.id
  status     = "active"
  scope_type = "dcim.site"
  scope_id   = netbox_site.fra1.id

  description = "Production vSphere cluster in Frankfurt"
  comments    = "Managed by Terraform."
}
