data "netbox_cooling_source" "ch01" {
  name = "CH-01"
}

# Cooling feed names are unique per cooling source.
data "netbox_cooling_feed" "a01_loop" {
  filters = [
    { name = "cooling_source_id", value = data.netbox_cooling_source.ch01.id },
    { name = "name", value = "CH-01/A01" },
  ]
}
