package provider

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// testAccProtoV6ProviderFactories serves the provider in-process to the
// Terraform CLI driven by terraform-plugin-testing.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"netbox": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccResourcePrefix prefixes every object created by acceptance tests so
// sweepers can find and delete leftovers.
const testAccResourcePrefix = "tfacc-"

// testAccPreCheck fails fast when the acceptance environment is incomplete.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, k := range []string{EnvServerURL, EnvAPIToken} {
		if os.Getenv(k) == "" {
			t.Fatalf("%s must be set for acceptance tests", k)
		}
	}
}

// testAccRandName returns a unique, sweeper-friendly object name.
func testAccRandName(t *testing.T) string {
	t.Helper()
	return testAccResourcePrefix + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
}

func TestAccRandNameHelper(t *testing.T) {
	a, b := testAccRandName(t), testAccRandName(t)
	if !strings.HasPrefix(a, testAccResourcePrefix) || a == b {
		t.Fatalf("testAccRandName produced %q and %q", a, b)
	}
}

// TestAccProviderConfigure verifies the provider can authenticate against the
// real NetBox named by the environment.
func TestAccProviderConfigure(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `provider "netbox" {}` + probeConfig,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.netbox_test_probe.t", "netbox_version"),
				resource.TestCheckResourceAttr("data.netbox_test_probe.t", "token_prefix", strings.Split(os.Getenv(EnvAPIToken), ".")[0]),
			),
		}},
	})
}
