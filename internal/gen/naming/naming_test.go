package naming

import "testing"

func TestGoField(t *testing.T) {
	cases := map[string]string{
		"custom_fields": "CustomFields", "is_pool": "IsPool", "_depth": "Depth", "_occupied": "Occupied",
		"vid_ranges": "VidRanges", "primary_ip4": "PrimaryIp4", "oob_ip": "OobIp", "l2vpn": "L2vpn",
		"ipsec_policy": "IpsecPolicy", "rf_channel": "RfChannel", "ssid": "Ssid", "a_terminations": "ATerminations",
		"tenant_group": "TenantGroup", "u_height": "UHeight", "http_method": "HttpMethod", "ui_visible": "UiVisible",
		"dcim_sites_partial_update": "DcimSitesPartialUpdate", "vpn_l2vpns_create": "VpnL2vpnsCreate",
		"ipam_ip_addresses_create": "IpamIpAddressesCreate", "name__ic": "NameIc", "asn__n": "AsnN",
	}
	for in, want := range cases {
		if got := GoField(in); got != want {
			t.Errorf("GoField(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGoType(t *testing.T) {
	cases := map[string]string{
		"IPAddress": "IPAddress", "VMInterface": "VMInterface", "L2VPN": "L2VPN", "ASNRange": "ASNRange",
		"IPSecProposal": "IPSecProposal", "BriefCircuitGroupAssignmentSerializer_Request": "BriefCircuitGroupAssignmentSerializerRequest",
		"WritableSiteRequest": "WritableSiteRequest",
	}
	for in, want := range cases {
		if got := GoType(in); got != want {
			t.Errorf("GoType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGoParam(t *testing.T) {
	if got := GoParam("type"); got != "type_" {
		t.Errorf("got %q", got)
	}
	if got := GoParam("device_type"); got != "deviceType" {
		t.Errorf("got %q", got)
	}
}

func TestSnakeSingular(t *testing.T) {
	cases := map[string]string{
		"ip-addresses": "ip_address", "sites": "site", "prefixes": "prefix", "ipsec-policies": "ipsec_policy",
		"virtual-chassis": "virtual_chassis", "asns": "asn", "interfaces": "interface", "rack-roles": "rack_role",
		"custom-field-choice-sets": "custom_field_choice_set", "aggregates": "aggregate", "l2vpns": "l2vpn",
		"module-bay-types": "module_bay_type", "virtual-machines": "virtual_machine",
	}
	for in, want := range cases {
		if got := Singular(Snake(in)); got != want {
			t.Errorf("Singular(Snake(%q)) = %q, want %q", in, got, want)
		}
	}
	if got := Snake("IPAddress"); got != "ip_address" {
		t.Errorf("Snake(IPAddress) = %q", got)
	}
	if got := Snake("VMInterface"); got != "vm_interface" {
		t.Errorf("Snake(VMInterface) = %q", got)
	}
}

func TestAttribute(t *testing.T) {
	cases := map[string]string{
		"ipaddresses": "ip_addresses", "ipaddress_count": "ip_address_count", "count_ipaddresses": "ip_address_count",
		"count_fhrp_groups": "fhrp_group_count", "virtualmachine_count": "virtual_machine_count",
		"devicetype_count": "device_type_count", "moduletype_count": "module_type_count",
		"inventoryitem_count": "inventory_item_count", "coolingfeed_count": "cooling_feed_count",
		"powerfeed_count": "power_feed_count", "wirelesslan_count": "wireless_lan_count",
		"terminations_count": "termination_count", "choices_count": "choice_count",
		"ip_addresses": "ip_addresses", "device_count": "device_count", "primary_ip4": "primary_ip4",
		"vm_role": "vm_role", "_depth": "_depth", "count": "count",
	}
	for in, want := range cases {
		if got := Attribute(in); got != want {
			t.Errorf("Attribute(%q) = %q, want %q", in, got, want)
		}
	}
}
