# A service is bound to a device or virtual machine and optionally to some of
# its IP addresses. Ports are expressed as "<protocol>/<port>" strings.
resource "netbox_service" "web" {
  parent_object_type = "dcim.device"
  parent_object_id   = netbox_device.web1.id
  name               = "nginx"
  port_mappings      = ["tcp/80", "tcp/443"]
  ipaddress_ids      = [netbox_ip_address.web1_eth0.id]
  description        = "Public web front end"
}

resource "netbox_service" "ssh" {
  parent_object_type = "virtualization.virtualmachine"
  parent_object_id   = netbox_virtual_machine.bastion.id
  name               = "sshd"
  port_mappings      = ["tcp/22"]
}
