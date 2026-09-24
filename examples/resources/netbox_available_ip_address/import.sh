# netbox_available_ip_address is imported by the ID of the allocated IP address
# (the number in the object's URL). The parent prefix_id / ip_range_id cannot be
# recovered and stay null; omit them from the configuration of imported resources.
terraform import netbox_available_ip_address.web01 123
