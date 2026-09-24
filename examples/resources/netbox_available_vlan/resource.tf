# Hand-maintained example for netbox_available_vlan.

resource "netbox_site" "hq" {
  name = "Headquarters"
  slug = "hq"
}

resource "netbox_vlan_group" "hq_access" {
  name       = "HQ access VLANs"
  slug       = "hq-access"
  scope_type = "dcim.site"
  scope_id   = netbox_site.hq.id
  vid_ranges = [[100, 199]]
}

resource "netbox_tenant" "finance" {
  name = "Finance"
  slug = "finance"
}

# Take the next free VID (100, 101, ...) from the group.
resource "netbox_available_vlan" "finance" {
  vlan_group_id = netbox_vlan_group.hq_access.id
  name          = "finance-users"
  status        = "active"
  tenant_id     = netbox_tenant.finance.id
  description   = "Finance department workstations"
}

output "finance_vid" {
  value = netbox_available_vlan.finance.vid
}
