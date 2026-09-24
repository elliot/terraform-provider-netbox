resource "netbox_rir" "rfc1918" {
  name        = "RFC 1918"
  slug        = "rfc-1918"
  is_private  = true
  description = "Private IPv4 address space"
}

resource "netbox_rir" "ripe" {
  name = "RIPE NCC"
  slug = "ripe-ncc"
}
