# netbox_available_vlan is imported by the ID of the allocated VLAN (the number
# in the object's URL). vlan_group_id is read back from the VLAN.
terraform import netbox_available_vlan.finance 123
