//go:build demo

package netbox

// Smoke test against a live NetBox (by default https://demo.netbox.dev).
//
//	set -a; source .env.demo; set +a; go test -tags demo ./netbox/ -run TestDemo -v

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestDemoSmoke(t *testing.T) {
	url, token := os.Getenv("NETBOX_SERVER_URL"), os.Getenv("NETBOX_API_TOKEN")
	if url == "" || token == "" {
		t.Skip("NETBOX_SERVER_URL / NETBOX_API_TOKEN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c := NewClient(url, &http.Client{Timeout: 60 * time.Second}, token)

	sites, err := ListAll(ctx, func(ctx context.Context, limit, offset int32) (Page[Site], *http.Response, error) {
		p, res, err := c.DcimAPI.DcimSitesList(ctx).Limit(limit).Offset(offset).Execute()
		if err != nil {
			return Page[Site]{}, res, err
		}
		return Page[Site]{Count: p.GetCount(), Results: p.GetResults()}, res, nil
	})
	if err != nil {
		t.Fatalf("list sites: %v", err)
	}
	t.Logf("sites: %d (first: %s status=%v)", len(sites), sites[0].GetName(), sites[0].GetStatus().Value.Get())

	name := fmt.Sprintf("tfacc-smoke-%d", time.Now().UnixNano()%1_000_000)
	req := NewTenantRequest(name, name)
	req.SetDescription("smoke")
	tenant, res, err := c.TenancyAPI.TenancyTenantsCreate(ctx).TenantRequest(*req).Execute()
	if err != nil {
		t.Fatalf("create tenant: %v", WrapError(err, res))
	}
	id := tenant.GetId()
	t.Cleanup(func() {
		if _, err := c.TenancyAPI.TenancyTenantsDestroy(context.Background(), id).Execute(); err != nil {
			t.Logf("cleanup: %v", err)
		}
	})

	patch := NewPatchedTenantRequest()
	patch.SetDescription("")
	patch.SetGroupNil()
	upd, res, err := c.TenancyAPI.TenancyTenantsPartialUpdate(ctx, id).PatchedTenantRequest(*patch).Execute()
	if err != nil {
		t.Fatalf("patch tenant: %v", WrapError(err, res))
	}
	if upd.GetDescription() != "" || upd.Group.IsSet() && upd.Group.Get() != nil {
		t.Fatalf("patch not applied: %+v", upd)
	}

	got, res, err := c.TenancyAPI.TenancyTenantsRetrieve(ctx, id).Execute()
	if err != nil {
		t.Fatalf("retrieve: %v", WrapError(err, res))
	}
	t.Logf("tenant %d custom_fields=%v tags=%v", got.GetId(), got.GetCustomFields(), got.GetTags())

	_, res, err = c.TenancyAPI.TenancyTenantsRetrieve(ctx, 2_000_000_000).Execute()
	if !IsNotFound(WrapError(err, res)) {
		t.Fatalf("expected 404, got %v", WrapError(err, res))
	}
}
