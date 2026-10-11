package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/elliot/terraform-provider-netbox/internal/gen/model"
)

func res(name string, attrs ...model.Attr) *model.Resource {
	return &model.Resource{Name: name, Plural: name + "s", Attrs: attrs}
}

func TestDiff(t *testing.T) {
	str := model.Attr{Name: "description", Kind: model.KindString}
	status := model.Attr{Name: "status", Kind: model.KindChoice, Enum: []string{"active", "planned"}}

	tests := []struct {
		name     string
		old, new []*model.Resource
		want     []Change
	}{
		{
			name: "identical",
			old:  []*model.Resource{res("site", str)},
			new:  []*model.Resource{res("site", str)},
		},
		{
			name: "new resource and skipped resources",
			old:  []*model.Resource{{Name: "bookmark", Skip: true}},
			new:  []*model.Resource{res("site"), {Name: "bookmark", Skip: true}},
			want: []Change{{Additive, "netbox_site", "", "new resource and data sources"}},
		},
		{
			name: "removed resource",
			old:  []*model.Resource{res("site")},
			want: []Change{{Breaking, "netbox_site", "", "removed"}},
		},
		{
			name: "resource became data source only",
			old:  []*model.Resource{res("job")},
			new:  []*model.Resource{{Name: "job", Plural: "jobs", DataSourceOnly: true}},
			want: []Change{{Breaking, "netbox_job", "", "resource removed; only the data sources remain"}},
		},
		{
			name: "attributes added",
			old:  []*model.Resource{res("site")},
			new: []*model.Resource{res("site",
				str,
				model.Attr{Name: "created", Kind: model.KindDateTime, ReadOnly: true},
				model.Attr{Name: "region_id", Kind: model.KindFK, Required: true},
			)},
			want: []Change{
				{Additive, "netbox_site", "created", "new computed attribute"},
				{Additive, "netbox_site", "description", "new optional attribute"},
				{Breaking, "netbox_site", "region_id", "new required attribute"},
			},
		},
		{
			name: "attribute removed and retyped",
			old:  []*model.Resource{res("site", str, model.Attr{Name: "asn", Kind: model.KindInt})},
			new:  []*model.Resource{res("site", model.Attr{Name: "asn", Kind: model.KindString})},
			want: []Change{
				{Breaking, "netbox_site", "asn", "type changed from int to string"},
				{Breaking, "netbox_site", "description", "attribute removed"},
			},
		},
		{
			name: "required and read-only flips",
			old: []*model.Resource{res("site",
				model.Attr{Name: "a", Kind: model.KindString},
				model.Attr{Name: "b", Kind: model.KindString, Required: true},
				model.Attr{Name: "c", Kind: model.KindString},
				model.Attr{Name: "d", Kind: model.KindString, RequiresReplace: true},
			)},
			new: []*model.Resource{res("site",
				model.Attr{Name: "a", Kind: model.KindString, Required: true},
				model.Attr{Name: "b", Kind: model.KindString},
				model.Attr{Name: "c", Kind: model.KindString, ReadOnly: true},
				model.Attr{Name: "d", Kind: model.KindString},
			)},
			want: []Change{
				{Breaking, "netbox_site", "a", "now required"},
				{Additive, "netbox_site", "b", "now optional"},
				{Breaking, "netbox_site", "c", "now read-only (computed only)"},
				{Additive, "netbox_site", "d", "changes no longer force replacement"},
			},
		},
		{
			name: "enum values",
			old:  []*model.Resource{res("site", status)},
			new: []*model.Resource{res("site",
				model.Attr{Name: "status", Kind: model.KindChoice, Enum: []string{"active", "retired"}})},
			want: []Change{
				{Breaking, "netbox_site", "status", "values removed: planned"},
				{Additive, "netbox_site", "status", "values added: retired"},
			},
		},
		{
			name: "nested list items",
			old: []*model.Resource{res("rack_type", model.Attr{Name: "units", Kind: model.KindNestedList,
				Nested: []model.Attr{{Name: "position", Kind: model.KindInt}}})},
			new: []*model.Resource{res("rack_type", model.Attr{Name: "units", Kind: model.KindNestedList,
				Nested: []model.Attr{{Name: "position", Kind: model.KindFloat}}})},
			want: []Change{{Breaking, "netbox_rack_type", "units.position", "type changed from int to float"}},
		},
		{
			name: "data source attributes and filters",
			old: []*model.Resource{{Name: "vlan", Plural: "vlans",
				ReadOnlyAttrs: []model.Attr{{Name: "display", Kind: model.KindString}},
				Filters:       []model.Filter{{Name: "site"}, {Name: "q"}}}},
			new: []*model.Resource{{Name: "vlan", Plural: "vlans",
				ReadOnlyAttrs: []model.Attr{{Name: "display_url", Kind: model.KindString}},
				Filters:       []model.Filter{{Name: "q"}, {Name: "related_to_site"}}}},
			want: []Change{
				{Breaking, "netbox_vlan", "display", "data source attribute removed"},
				{Additive, "netbox_vlan", "display_url", "new data source attribute"},
				{Breaking, "netbox_vlans", "filters", "filters removed: site"},
				{Additive, "netbox_vlans", "filters", "filters added: related_to_site"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Diff(tt.old, tt.new)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d changes, want %d:\n%v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("change %d:\n got  %+v\n want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestWriteMarkdown(t *testing.T) {
	var buf bytes.Buffer
	writeMarkdown(&buf, Verdict{OldVersion: "4.7.0", NewVersion: "4.7.2"})
	if !strings.Contains(buf.String(), "No change") {
		t.Errorf("empty diff: %q", buf.String())
	}

	buf.Reset()
	writeMarkdown(&buf, Verdict{OldVersion: "4.7.0", NewVersion: "4.8.0", Breaking: 1, Additive: 1, Changes: []Change{
		{Additive, "netbox_site", "tags", "new optional attribute"},
		{Breaking, "netbox_site", "", "a | b"},
	}})
	out := buf.String()
	if strings.Index(out, "| breaking |") > strings.Index(out, "| additive |") {
		t.Errorf("breaking changes must be listed first:\n%s", out)
	}
	if !strings.Contains(out, `a \| b`) {
		t.Errorf("pipes must be escaped:\n%s", out)
	}
}
