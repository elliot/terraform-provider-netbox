# netbox_device_primary_ip is imported by the device ID. The primary IPv4 is
# picked when set, otherwise the primary IPv6.
terraform import netbox_device_primary_ip.leaf1 42
