# Rack reservations have no natural key; look them up by ID or by filters.
data "netbox_rack_reservation" "by_id" {
  id = 123
}

data "netbox_rack" "a01" {
  name = "A01"
}

data "netbox_rack_reservation" "a01_storage" {
  filters = [
    { name = "rack_id", value = data.netbox_rack.a01.id },
    { name = "description", value = "Reserved for storage array delivery in Q3" },
  ]
}
