package render

import (
	"strings"
	"testing"

	"github.com/elliot/terraform-provider-netbox/internal/gen/model"
)

func TestUpgradeTest(t *testing.T) {
	r := &model.Resource{App: "dcim", Name: "site", GoName: "Site", Path: "/api/dcim/sites/"}
	tests := []struct {
		name      string
		test      model.Test
		skip      string
		hasUpdate bool
		want      []string
		notWant   []string
	}{
		{
			name:      "update then basic",
			hasUpdate: true,
			want:      []string{"func TestAccSite_upgrade(", "acctest.Render(t, siteTestConfigUpdate, name),\n\t\t\tacctest.Render(t, siteTestConfigBasic, name),"},
			notWant:   []string{"t.Skip", "Serial"},
		},
		{
			name:    "basic only",
			want:    []string{"acctest.Render(t, siteTestConfigBasic, name),\n\t\t},"},
			notWant: []string{"siteTestConfigUpdate"},
		},
		{
			name:      "override config and serial",
			test:      model.Test{UpgradeConfig: "x", Serial: true},
			hasUpdate: true,
			want:      []string{"acctest.Render(t, siteTestConfigUpgrade, name),\n\t\t},", "Serial: true,"},
		},
		{
			name: "upgrade skip",
			test: model.Test{UpgradeSkip: "renamed in 1.0"},
			want: []string{`t.Skip("renamed in 1.0")`},
		},
		{
			name: "basic test skip wins",
			test: model.Test{UpgradeSkip: "other"},
			skip: "no fixture",
			want: []string{`t.Skip("no fixture")`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := *r
			rr.Test = tt.test
			got := upgradeTest(&rr, tt.skip, tt.hasUpdate)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("missing %q in:\n%s", w, got)
				}
			}
			for _, w := range tt.notWant {
				if strings.Contains(got, w) {
					t.Errorf("unexpected %q in:\n%s", w, got)
				}
			}
		})
	}
}
