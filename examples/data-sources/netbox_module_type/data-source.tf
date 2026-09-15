data "netbox_module_type" "nm_8x" {
  filters = [
    { name = "manufacturer", value = "cisco" },
    { name = "model", value = "C9300-NM-8X" },
  ]
}
