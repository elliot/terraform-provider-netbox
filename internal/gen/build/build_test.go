package build

import (
	"strings"
	"testing"

	"github.com/elliot/terraform-provider-netbox/internal/gen/model"
)

func TestCheckSymmetricPairs(t *testing.T) {
	pair := func(asnSitesReadOnly bool) []*model.Resource {
		return []*model.Resource{
			{Name: "asn", Attrs: []model.Attr{{Name: "site_ids", Kind: model.KindFKList, Target: "site", ReadOnly: asnSitesReadOnly}}},
			{Name: "site", Attrs: []model.Attr{{Name: "asn_ids", Kind: model.KindFKList, Target: "asn"}}},
			// One-sided relations and self-references are fine.
			{Name: "vrf", Attrs: []model.Attr{{Name: "import_target_ids", Kind: model.KindFKList, Target: "route_target"}}},
			{Name: "route_target"},
			{Name: "interface", Attrs: []model.Attr{{Name: "bridge_ids", Kind: model.KindFKList, Target: "interface"}}},
		}
	}
	index := func(rs []*model.Resource) map[string]*model.Resource {
		m := map[string]*model.Resource{}
		for _, r := range rs {
			m[r.Name] = r
		}
		return m
	}

	rs := pair(true)
	if err := checkSymmetricPairs(rs, index(rs)); err != nil {
		t.Fatalf("read-only reverse side: unexpected error: %v", err)
	}

	rs = pair(false)
	err := checkSymmetricPairs(rs, index(rs))
	if err == nil {
		t.Fatal("two writable sides: expected an error")
	}
	if got := err.Error(); strings.Count(got, "\n") != 0 || !strings.Contains(got, "netbox_asn.site_ids and netbox_site.asn_ids") {
		t.Fatalf("expected the pair to be reported once, got:\n%s", got)
	}

	rs = pair(false)
	rs[1].Skip = true
	if err := checkSymmetricPairs(rs, index(rs)); err != nil {
		t.Fatalf("skipped resource: unexpected error: %v", err)
	}
}
