package upgrade_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/elliot/terraform-provider-netbox/internal/provider"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/manual"
	"github.com/elliot/terraform-provider-netbox/internal/upgrade"
)

// TestEveryVersionedResourceHasUpgraders checks every resource of the
// provider: a schema version above 0 needs a state upgrader from each older
// version in the Default registry, and upgraders registered for a type must
// match a resource and its current version.
func TestEveryVersionedResourceHasUpgraders(t *testing.T) {
	ctx := context.Background()
	p := provider.New("test")()
	seen := map[string]bool{}
	for _, f := range p.Resources(ctx) {
		r := f()
		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "netbox"}, &meta)
		var sch resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &sch)
		version := sch.Schema.GetVersion()
		seen[meta.TypeName] = true
		if version > 0 {
			if _, ok := r.(resource.ResourceWithUpgradeState); !ok {
				t.Errorf("%s has schema version %d but does not implement UpgradeState", meta.TypeName, version)
			}
		}
		if err := upgrade.Default.Validate(meta.TypeName, version); err != nil {
			t.Error(err)
		}
	}
	for _, typ := range upgrade.Default.Types() {
		if !seen[typ] {
			t.Errorf("state upgraders registered for %s, which is not a resource of the provider", typ)
		}
	}
}
