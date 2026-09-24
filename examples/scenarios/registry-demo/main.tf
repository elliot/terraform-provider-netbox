# Registry smoke test against the public demo.
#
# Unlike the other scenarios, which were validated with a locally built
# provider (dev_overrides), this one installs the provider from the public
# Terraform Registry. It proves that a published release:
#   * downloads, verifies (GPG-signed SHA256SUMS) and starts on this platform,
#   * configures against a live NetBox 4.x (v2 token, /api/status/ check),
#   * creates, reads, updates and deletes objects through the common paths:
#     plain resources, foreign keys, tags, native custom_fields, an allocation
#     resource, a single-object data source and a list data source,
#   * converges (the second plan is empty) and imports by ID.
#
# Run it with scripts/registry-smoke.sh (or `make registry-smoke`), which
# provisions a demo token, isolates Terraform from dev_overrides and local
# mirrors, and runs apply -> plan -> import -> update -> destroy. By hand:
#
#   make demo-token && set -a && . ./.env.demo && set +a
#   terraform init && terraform apply && terraform plan && terraform destroy
#
# Every object is prefixed so leftovers can be swept (`make sweep` removes
# tfacc-* objects). The demo database is reset daily.

terraform {
  required_version = ">= 1.12"

  required_providers {
    netbox = {
      source  = "elliot/netbox"
      version = "~> 0.1"
    }
  }
}

provider "netbox" {
  # server_url and api_token come from NETBOX_SERVER_URL / NETBOX_API_TOKEN.
  requests_per_second = 5 # be polite to the shared demo instance
}

variable "prefix" {
  description = "Prefix for every object name and slug created by this scenario."
  type        = string
  default     = "tfacc-registry"
}

variable "network" {
  description = "IPv4 /24 used for the test prefix. The default is benchmarking space (RFC 2544) that the demo data does not use."
  type        = string
  default     = "198.18.42.0/24"
}

variable "description" {
  description = "Description set on the site; the smoke script changes it to exercise an in-place update."
  type        = string
  default     = "Created by the registry smoke test"
}

locals {
  # Custom field names must match ^[a-z0-9_]+$.
  cf_name = "${replace(var.prefix, "-", "_")}_owner"
}

# --------------------------------------------------------------------------
# Plain resources, tags and foreign keys
# --------------------------------------------------------------------------

resource "netbox_tag" "smoke" {
  name  = "${var.prefix}-managed"
  slug  = "${var.prefix}-managed"
  color = "2196f3"
}

resource "netbox_tenant" "smoke" {
  name = "${var.prefix}-tenant"
  slug = "${var.prefix}-tenant"
  tags = [netbox_tag.smoke.slug]
}

# --------------------------------------------------------------------------
# Native custom_fields object
# --------------------------------------------------------------------------

resource "netbox_custom_field" "owner" {
  name         = local.cf_name
  label        = "Owner (registry smoke)"
  type         = "text"
  object_types = ["dcim.site"]
}

resource "netbox_site" "smoke" {
  name        = "${var.prefix}-site"
  slug        = "${var.prefix}-site"
  status      = "active"
  description = var.description
  tenant_id   = netbox_tenant.smoke.id
  tags        = [netbox_tag.smoke.slug]

  custom_fields = {
    (local.cf_name) = "network-team"
  }

  # The field must exist before a value can be set, and must outlive it on destroy.
  depends_on = [netbox_custom_field.owner]
}

# --------------------------------------------------------------------------
# IPAM: scoped prefix and an allocated address
# --------------------------------------------------------------------------

resource "netbox_prefix" "smoke" {
  prefix      = var.network
  status      = "active"
  scope_type  = "dcim.site"
  scope_id    = netbox_site.smoke.id
  tenant_id   = netbox_tenant.smoke.id
  description = "${var.prefix} prefix"
  tags        = [netbox_tag.smoke.slug]
}

resource "netbox_available_ip_address" "gateway" {
  prefix_id   = netbox_prefix.smoke.id
  status      = "active"
  dns_name    = "gw.${var.prefix}.example"
  description = "${var.prefix} gateway"
  tenant_id   = netbox_tenant.smoke.id
}

# --------------------------------------------------------------------------
# Data sources (single object by attribute, list by generic filter)
# --------------------------------------------------------------------------

data "netbox_site" "by_slug" {
  slug = netbox_site.smoke.slug
}

data "netbox_prefixes" "by_tenant" {
  filters = [{ name = "tenant_id", value = tostring(netbox_tenant.smoke.id) }]

  depends_on = [netbox_prefix.smoke]
}

# --------------------------------------------------------------------------
# Assertions: fail the plan/apply if the provider reads back something else
# --------------------------------------------------------------------------

check "site_round_trip" {
  assert {
    condition     = data.netbox_site.by_slug.id == netbox_site.smoke.id
    error_message = "The site data source did not resolve to the managed site."
  }
  assert {
    condition     = jsondecode(data.netbox_site.by_slug.custom_fields)[local.cf_name] == "network-team"
    error_message = "The custom field value was not read back by the site data source."
  }
}

check "prefix_list" {
  assert {
    condition     = length(data.netbox_prefixes.by_tenant.items) == 1
    error_message = "Expected exactly one prefix for the smoke tenant."
  }
}

check "allocation" {
  assert {
    # The first free host of an empty prefix: .1 for anything but a /31 or /32.
    condition     = netbox_available_ip_address.gateway.address == "${cidrhost(var.network, 1)}/${split("/", var.network)[1]}"
    error_message = "The allocated address is not the first host of the smoke prefix."
  }
}

output "site_id" {
  value = netbox_site.smoke.id
}

output "tag_id" {
  description = "Used by the smoke script for the import round-trip."
  value       = netbox_tag.smoke.id
}

output "gateway_address" {
  value = netbox_available_ip_address.gateway.address
}

output "site_custom_fields" {
  value = jsondecode(data.netbox_site.by_slug.custom_fields)
}
