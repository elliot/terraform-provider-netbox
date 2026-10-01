package manual_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/elliot/terraform-provider-netbox/internal/acctest"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
)

const customFieldForeignDefaultConfig = `resource "netbox_custom_field" "test" {
  name         = "{{.Snake}}"
  type         = "text"
  object_types = ["dcim.site"]
  description  = "{{.Name}}"
}
`

// TestAccCustomField_foreignEmptyDefault covers objects written by other tools
// (e-breuninger/netbox sends custom field defaults as plain strings), which
// store "" in nullable JSON columns where NetBox itself leaves NULL. Leaving
// the attribute out of the configuration must clear it with an explicit null
// rather than fail with "inconsistent result after apply", both after an
// import and after the value reappears out of band.
func TestAccCustomField_foreignEmptyDefault(t *testing.T) {
	name := acctest.RandName()
	rn := "netbox_custom_field.test"
	cfg := acctest.Render(t, customFieldForeignDefaultConfig, name)
	var id int64

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:             acctest.CheckDestroyed("netbox_custom_field", "/api/extras/custom-fields/"),
		Steps: []resource.TestStep{
			{
				// Import a custom field created outside Terraform with default "".
				PreConfig: func() {
					var out struct {
						ID int64 `json:"id"`
					}
					foreignWrite(t, "POST", "/api/extras/custom-fields/", map[string]any{
						"name":         strings.ReplaceAll(name, "-", "_"),
						"type":         "text",
						"object_types": []string{"dcim.site"},
						"description":  name,
						"default":      "",
					}, &out)
					id = out.ID
				},
				Config:             cfg,
				ResourceName:       rn,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc:  func(*terraform.State) (string, error) { return strconv.FormatInt(id, 10), nil },
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if got := states[0].Attributes["default"]; got != `""` {
						return fmt.Errorf("imported default = %q, want the JSON string %q", got, `""`)
					}
					return nil
				},
			},
			{
				Config: cfg,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(rn, "default"),
					checkCustomFieldDefaultNull(&id),
				),
			},
			{
				// Another tool writes "" again: refresh shows the drift and the
				// apply clears it.
				PreConfig: func() {
					foreignWrite(t, "PATCH", fmt.Sprintf("/api/extras/custom-fields/%d/", id), map[string]any{"default": ""}, nil)
				},
				Config: cfg,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(rn, "default"),
					checkCustomFieldDefaultNull(&id),
				),
			},
		},
	})
}

// foreignWrite sends a raw API request, standing in for a tool other than this
// provider.
func foreignWrite(t *testing.T, method, apiPath string, body map[string]any, out any) {
	t.Helper()
	c, err := acctest.Client()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	switch method {
	case "POST":
		err = c.PostRaw(ctx, apiPath, body, out)
	case "PATCH":
		err = c.PatchRaw(ctx, apiPath, body, out)
	default:
		t.Fatalf("unsupported method %s", method)
	}
	if err != nil {
		t.Fatalf("%s %s: %v", method, apiPath, err)
	}
}

// checkCustomFieldDefaultNull asserts that NetBox stores NULL (not "") as the
// default of the custom field.
func checkCustomFieldDefaultNull(id *int64) resource.TestCheckFunc {
	return func(*terraform.State) error {
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		var out map[string]any
		if err := c.GetRaw(context.Background(), fmt.Sprintf("/api/extras/custom-fields/%d/", *id), &out); err != nil {
			return err
		}
		if v, ok := out["default"]; !ok || v != nil {
			return fmt.Errorf("NetBox default = %#v, want null", v)
		}
		return nil
	}
}
