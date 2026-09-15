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
