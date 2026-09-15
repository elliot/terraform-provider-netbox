data "netbox_wireless_link" "backhaul" {
  filters = [{ name = "ssid", value = "backhaul-a-b" }]
}

output "backhaul_distance_km" {
  value = data.netbox_wireless_link.backhaul.distance
}
