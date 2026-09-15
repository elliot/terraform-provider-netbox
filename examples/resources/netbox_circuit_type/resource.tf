resource "netbox_circuit_type" "example" {
  name        = "Internet access"
  slug        = "internet-access"
  color       = "2196f3"
  description = "Dedicated internet access circuits"
}
