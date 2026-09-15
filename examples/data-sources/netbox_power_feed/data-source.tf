# Power feed names are unique per power panel.
data "netbox_power_feed" "a01_a" {
  filters = [
    { name = "power_panel", value = "PP-A" },
    { name = "name", value = "A01-A" },
  ]
}

output "feed_voltage" {
  value = data.netbox_power_feed.a01_a.voltage
}
