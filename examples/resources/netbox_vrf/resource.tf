resource "netbox_route_target" "acme_import" {
  name = "65000:100"
}

resource "netbox_route_target" "acme_export" {
  name = "65000:101"
}

resource "netbox_vrf" "acme" {
  name              = "ACME-L3VPN"
  rd                = "65000:100"
  enforce_unique    = true
  import_target_ids = [netbox_route_target.acme_import.id]
  export_target_ids = [netbox_route_target.acme_export.id]
  description       = "Customer L3VPN"
}
