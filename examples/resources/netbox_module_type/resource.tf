resource "netbox_manufacturer" "cisco" {
  name = "Cisco"
  slug = "cisco"
}
resource "netbox_module_type" "nm_8x" {
  manufacturer_id = netbox_manufacturer.cisco.id
  model           = "C9300-NM-8X"
  part_number     = "C9300-NM-8X"
  airflow         = "front-to-rear"
  weight          = 0.45
  weight_unit     = "kg"
  description     = "8x 10G SFP+ network module"
}
# Interface templates on the module type are replicated onto the device when
# a module of this type is installed; {module} becomes the bay position.
resource "netbox_interface_template" "uplinks" {
  count          = 8
  module_type_id = netbox_module_type.nm_8x.id
  name           = "TenGigabitEthernet1/{module}/${count.index + 1}"
  type           = "10gbase-x-sfpp"
}
