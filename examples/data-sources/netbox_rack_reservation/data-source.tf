# Rack reservations have no natural key; look them up by ID or by filters.
data "netbox_rack_reservation" "by_id" {
  id = 123
}

data "netbox_rack_reservation" "a01_storage" {
  filters = [
    { name = "rack", value = "A01" },
    { name = "description", value = "Reserved for storage array delivery in Q3" },
  ]
}
