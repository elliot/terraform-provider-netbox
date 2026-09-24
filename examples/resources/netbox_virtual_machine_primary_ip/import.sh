# netbox_virtual_machine_primary_ip is imported by the virtual machine ID. The
# primary IPv4 is picked when set, otherwise the primary IPv6.
terraform import netbox_virtual_machine_primary_ip.app01 42
