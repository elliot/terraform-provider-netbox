# netbox_available_prefix is imported by the ID of the allocated prefix (the
# number in the object's URL). parent_prefix_id and prefix_length cannot be
# recovered and stay null; omit them from the configuration of imported resources.
terraform import netbox_available_prefix.hq_users 123
