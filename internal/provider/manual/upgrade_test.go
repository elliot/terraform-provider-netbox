package manual_test

import (
	"testing"

	"github.com/elliot/terraform-provider-netbox/internal/acctest"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
)

// Upgrade tests for the hand-written resources: apply the fixture with the
// release named by NETBOX_UPGRADE_FROM, then expect an empty plan from this
// build. Each tries the update fixture first and falls back to the basic one
// when the release lacks an attribute (see acctest.UpgradeTest).

func TestAccAvailablePrefix_upgrade(t *testing.T) {
	name := acctest.RandName()
	o := octet(name)
	acctest.UpgradeTest(t, acctest.UpgradeCase{
		ResourceType: "netbox_available_prefix",
		APIPath:      "/api/ipam/prefixes/",
		Configs:      []string{render(t, availablePrefixConfigUpdate, name, o), render(t, availablePrefixConfigBasic, name, o)},
	})
}

func TestAccAvailableIpAddress_upgrade(t *testing.T) {
	name := acctest.RandName()
	o := octet(name)
	acctest.UpgradeTest(t, acctest.UpgradeCase{
		ResourceType: "netbox_available_ip_address",
		APIPath:      "/api/ipam/ip-addresses/",
		Configs:      []string{render(t, availableIpAddressConfigUpdate, name, o), render(t, availableIpAddressConfigBasic, name, o)},
	})
}

func TestAccAvailableAsn_upgrade(t *testing.T) {
	name := acctest.RandName()
	start := 4200000000 + (hash(name)%900000)*100
	end := start + 9
	acctest.UpgradeTest(t, acctest.UpgradeCase{
		ResourceType: "netbox_available_asn",
		APIPath:      "/api/ipam/asns/",
		Configs:      []string{render(t, availableAsnConfigUpdate, name, start, end), render(t, availableAsnConfigBasic, name, start, end)},
	})
}

func TestAccAvailableVlan_upgrade(t *testing.T) {
	name := acctest.RandName()
	acctest.UpgradeTest(t, acctest.UpgradeCase{
		ResourceType: "netbox_available_vlan",
		APIPath:      "/api/ipam/vlans/",
		Configs:      []string{acctest.Render(t, availableVlanConfigUpdate, name), acctest.Render(t, availableVlanConfigBasic, name)},
	})
}

func TestAccDevicePrimaryIp_upgrade(t *testing.T) {
	name := acctest.RandName()
	o := octet(name)
	acctest.UpgradeTest(t, acctest.UpgradeCase{
		ResourceType: "netbox_device_primary_ip",
		APIPath:      "/api/dcim/devices/",
		Configs:      []string{render(t, devicePrimaryIpConfigUpdate, name, o)},
	})
}

func TestAccVirtualMachinePrimaryIp_upgrade(t *testing.T) {
	name := acctest.RandName()
	o := octet(name)
	acctest.UpgradeTest(t, acctest.UpgradeCase{
		ResourceType: "netbox_virtual_machine_primary_ip",
		APIPath:      "/api/virtualization/virtual-machines/",
		Configs:      []string{render(t, virtualMachinePrimaryIpConfigUpdate, name, o), render(t, virtualMachinePrimaryIpConfigBasic, name, o)},
	})
}

func TestAccCustomFields_upgrade(t *testing.T) {
	name := acctest.RandName()
	acctest.UpgradeTest(t, acctest.UpgradeCase{
		ResourceType: "netbox_tenant",
		APIPath:      "/api/tenancy/tenants/",
		Configs:      []string{acctest.Render(t, customFieldsAllTypesConfig, name)},
		Serial:       true, // custom fields are global on the shared demo
	})
}

func TestAccCustomFieldValue_upgrade(t *testing.T) {
	name := acctest.RandName()
	acctest.UpgradeTest(t, acctest.UpgradeCase{
		ResourceType: "netbox_site",
		APIPath:      "/api/dcim/sites/",
		Configs:      []string{render(t, customFieldValueConfig, name, "team-a")},
		Serial:       true,
	})
}
