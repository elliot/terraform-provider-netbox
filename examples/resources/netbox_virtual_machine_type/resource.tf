# Virtual machine types (NetBox 4.7+) are reusable size profiles for VMs.
resource "netbox_virtual_machine_type" "m_large" {
  name        = "m.large"
  slug        = "m-large"
  description = "2 vCPU, 8 GB RAM, general purpose"
}
