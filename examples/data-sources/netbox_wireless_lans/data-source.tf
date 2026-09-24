data "netbox_wireless_lans" "campus" {
  filters = [
    { name = "group", value = "campus" },
    { name = "status", value = "active" },
  ]
}

output "campus_ssids" {
  value = data.netbox_wireless_lans.campus.items[*].ssid
}
