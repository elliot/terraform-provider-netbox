package manual_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/elliot/terraform-provider-netbox/internal/acctest"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
)

const siteCoordinatesConfig = `resource "netbox_site" "test" {
  name      = "{{.Name}}"
  slug      = "{{.Name}}"
  latitude  = %s
  longitude = %s
}
`

// TestAccSite_coordinatePrecision checks that GPS coordinates are limited to
// the six decimals (xx.yyyyyy) NetBox stores: anything finer is rejected at
// plan time instead of being rounded by the server into a diff.
func TestAccSite_coordinatePrecision(t *testing.T) {
	name := acctest.RandName()
	rn := "netbox_site.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:             acctest.CheckDestroyed("netbox_site", "/api/dcim/sites/"),
		Steps: []resource.TestStep{
			{
				Config:      acctest.Render(t, siteCoordinatesFmt("52.5200081", "13.404954"), name),
				ExpectError: regexp.MustCompile(`(?s)Too many decimal places.*52\.5200081`),
			},
			{
				Config:      acctest.Render(t, siteCoordinatesFmt("52.520008", "13.4049541"), name),
				ExpectError: regexp.MustCompile(`(?s)Too many decimal places.*13\.4049541`),
			},
			{
				// Six decimals and fewer round-trip exactly (the post-apply
				// plan must be empty, which the harness asserts).
				Config: acctest.Render(t, siteCoordinatesFmt("-33.865143", "151.2099"), name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "latitude", "-33.865143"),
					resource.TestCheckResourceAttr(rn, "longitude", "151.2099"),
				),
			},
		},
	})
}

func siteCoordinatesFmt(lat, lon string) string {
	return fmt.Sprintf(siteCoordinatesConfig, lat, lon)
}
