// Package acctest holds shared helpers for acceptance tests of generated and
// hand-written resources. It is imported by _test.go files only.
package acctest

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/elliot/terraform-provider-netbox/internal/provider"
	"github.com/elliot/terraform-provider-netbox/netbox"
)

// Prefix prefixes every object created by acceptance tests so sweepers can
// find and delete leftovers.
const Prefix = "tfacc-"

// ProviderFactories serves the provider in-process to the Terraform CLI.
var ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"netbox": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// PreCheck fails fast when the acceptance environment is incomplete.
func PreCheck(t *testing.T) {
	t.Helper()
	for _, k := range []string{provider.EnvServerURL, provider.EnvAPIToken} {
		if os.Getenv(k) == "" {
			t.Fatalf("%s must be set for acceptance tests", k)
		}
	}
}

// RandName returns a unique, sweeper-friendly object name.
func RandName() string {
	return Prefix + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
}

// Render executes an HCL text/template with {{.Name}} set to name and the
// provider block prepended.
func Render(t *testing.T, tmpl, name string) string {
	t.Helper()
	tp, err := template.New("cfg").Parse(tmpl)
	if err != nil {
		t.Fatalf("bad test config template: %v", err)
	}
	var buf bytes.Buffer
	if err := tp.Execute(&buf, map[string]string{"Name": name, "Prefix": Prefix}); err != nil {
		t.Fatalf("render test config: %v", err)
	}
	return "provider \"netbox\" {}\n\n" + buf.String()
}

var (
	clientOnce sync.Once
	client     *netbox.APIClient
	clientErr  error
)

// Client returns a shared API client built from the environment (for
// CheckDestroy and sweepers).
func Client() (*netbox.APIClient, error) {
	clientOnce.Do(func() {
		url, token := os.Getenv(provider.EnvServerURL), os.Getenv(provider.EnvAPIToken)
		if url == "" || token == "" {
			clientErr = fmt.Errorf("%s and %s must be set", provider.EnvServerURL, provider.EnvAPIToken)
			return
		}
		client = netbox.NewClient(url, &http.Client{Timeout: 60 * time.Second}, token)
	})
	return client, clientErr
}

// CheckDestroyed returns a CheckDestroy function asserting that every instance
// of the given resource type in state no longer exists at apiPath + id.
func CheckDestroyed(resourceType, apiPath string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c, err := Client()
		if err != nil {
			return err
		}
		for _, rs := range s.RootModule().Resources {
			if rs.Type != resourceType {
				continue
			}
			id := rs.Primary.ID
			if id == "" {
				id = rs.Primary.Attributes["id"]
			}
			var out map[string]any
			err := c.GetRaw(context.Background(), strings.TrimSuffix(apiPath, "/")+"/"+id+"/", &out)
			if err == nil {
				return fmt.Errorf("%s %s still exists", resourceType, id)
			}
			if !netbox.IsNotFound(err) {
				return fmt.Errorf("%s %s: unexpected error: %w", resourceType, id, err)
			}
		}
		return nil
	}
}

// Sweep deletes every object under apiPath whose search matches the test
// prefix. It queries `q=<prefix>` and, when supported, `description__ic=<prefix>`.
func Sweep(apiPath string, filters []string) error {
	c, err := Client()
	if err != nil {
		return err
	}
	ctx := context.Background()
	seen := map[int64]bool{}
	for _, f := range filters {
		offset := 0
		for {
			var page struct {
				Count   int              `json:"count"`
				Results []map[string]any `json:"results"`
			}
			url := fmt.Sprintf("%s?%s=%s&limit=200&offset=%d", apiPath, f, Prefix, offset)
			if err := c.GetRaw(ctx, url, &page); err != nil {
				// Unsupported filter: try the next one.
				break
			}
			for _, r := range page.Results {
				idf, _ := r["id"].(float64)
				id := int64(idf)
				if id == 0 || seen[id] {
					continue
				}
				seen[id] = true
				if err := c.DeleteRaw(ctx, fmt.Sprintf("%s%d/", apiPath, id)); err != nil && !netbox.IsNotFound(err) {
					return fmt.Errorf("delete %s%d/: %w", apiPath, id, err)
				}
			}
			offset += len(page.Results)
			if len(page.Results) == 0 || offset >= page.Count {
				break
			}
		}
	}
	return nil
}
