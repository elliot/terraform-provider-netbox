package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/elliot/terraform-provider-netbox/internal/client"
)

// TestMain enables `go test ./internal/provider -sweep=all` to run the
// registered sweepers.
func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func init() {
	// A connectivity sweeper other sweepers can depend on; it fails early
	// when the environment is not usable.
	resource.AddTestSweepers("netbox_connectivity", &resource.Sweeper{
		Name: "netbox_connectivity",
		F: func(_ string) error {
			pd, err := sharedHTTPClient()
			if err != nil {
				return err
			}
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, pd.ServerURL+"/api/status/", nil)
			if err != nil {
				return err
			}
			resp, err := pd.HTTPClient.Do(req)
			if err != nil {
				return err
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("GET /api/status/ returned %d", resp.StatusCode)
			}
			return nil
		},
	})
}

// sharedHTTPClient builds a ProviderData from the environment for sweepers.
// The generated API client (ProviderData.API) is left nil for now.
func sharedHTTPClient() (*ProviderData, error) {
	s, diags := resolveSettings(context.Background(), providerModel{}, "sweeper", os.Getenv)
	if diags.HasError() {
		return nil, fmt.Errorf("resolving provider settings from environment: %v", diags.Errors())
	}
	hc, err := client.NewHTTPClient(s.client)
	if err != nil {
		return nil, err
	}
	return &ProviderData{
		HTTPClient: hc,
		ServerURL:  s.client.ServerURL,
		Token:      s.client.Token,
		UserAgent:  s.client.UserAgent,
	}, nil
}
