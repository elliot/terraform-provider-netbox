# Cooling feed names are unique per cooling source.
data "netbox_cooling_feed" "a01_loop" {
  filters = [
    { name = "cooling_source", value = "CH-01" },
    { name = "name", value = "CH-01/A01" },
  ]
}
