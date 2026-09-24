# netbox_available_asn is imported by the ID of the allocated ASN object (the
# number in the object's URL, not the AS number). asn_range_id cannot be
# recovered and stays null; omit it from the configuration of imported resources.
terraform import netbox_available_asn.acme_edge 123
