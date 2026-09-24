data "netbox_data_source" "example" {
  name = "config-templates"
}

# Any API filter of /api/core/data-sources/ works; the lookup must match exactly one object.
data "netbox_data_source" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "data_source_id" {
  value = data.netbox_data_source.example.id
}
