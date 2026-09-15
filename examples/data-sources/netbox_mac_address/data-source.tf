# MAC addresses are not unique in NetBox; look one up by ID or narrow with filters.
data "netbox_mac_address" "by_id" {
  id = 123
}

data "netbox_mac_address" "sw01_eth0" {
  filters = [
    { name = "mac_address", value = "00:1B:44:11:3A:B7" },
    { name = "device", value = "fra1-sw01" },
  ]
}
