data "netbox_virtual_circuit" "pw9001" {
  filters = [
    { name = "cid", value = "LUM-L2VPN-9001" },
  ]
}

output "pw9001_status" {
  value = data.netbox_virtual_circuit.pw9001.status
}
