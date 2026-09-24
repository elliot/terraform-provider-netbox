# A git repository synchronised by NetBox (config templates, export
# templates, scripts ...). Sync it with the NetBox UI or the API.
resource "netbox_data_source" "templates" {
  name          = "config-templates"
  type          = "git"
  source_url    = "https://github.com/example/netbox-templates.git"
  enabled       = true
  sync_interval = 1440
  ignore_rules  = "*.md\nREADME*"
  parameters    = jsonencode({ branch = "main" })
  description   = "Device configuration templates"
}

resource "netbox_data_source" "local" {
  name       = "local-scripts"
  type       = "local"
  source_url = "file:///opt/netbox/scripts"
}
