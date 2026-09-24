data "netbox_wireless_lan" "guest" {
  filters = [{ name = "ssid", value = "Guest-WiFi" }]
}

output "guest_vlan_id" {
  value = data.netbox_wireless_lan.guest.vlan_id
}
