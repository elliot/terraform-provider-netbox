resource "netbox_service_template" "https" {
  name          = "HTTPS"
  port_mappings = ["tcp/443"]
  description   = "Web front end"
}

resource "netbox_service_template" "dns" {
  name          = "DNS"
  port_mappings = ["udp/53", "tcp/53"]
}
