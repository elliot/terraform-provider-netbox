data "netbox_custom_link" "weather" {
  name = "weather"
}

output "weather_link_url" {
  value = data.netbox_custom_link.weather.link_url
}
