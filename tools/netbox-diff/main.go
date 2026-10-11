// Command netbox-diff compares the Terraform surface the generator builds from
// two NetBox OpenAPI documents and classifies every difference as breaking or
// additive, so a NetBox release can be judged by what it changes in the
// provider rather than by the raw spec diff (which is never empty and mostly
// noise: descriptions, enum labels, schemas the provider does not use).
//
//	netbox-diff -old spec/netbox-4.7.0.openapi.json -new spec/netbox-4.7.2.openapi.json \
//	    [-overrides generator/overrides] [-markdown report.md] [-json verdict.json]
//
// Both documents must sit next to a required-fixes.json (put the new one in
// spec/), because the generator reads the corrections from the document's
// directory and silently skips them when the file is missing.
//
// Used by .github/workflows/netbox-watch.yml; see docs/DEVELOPMENT.md.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/elliot/terraform-provider-netbox/internal/gen/build"
	"github.com/elliot/terraform-provider-netbox/internal/gen/model"
	"github.com/elliot/terraform-provider-netbox/internal/gen/openapi"
	"github.com/elliot/terraform-provider-netbox/internal/gen/overrides"
)

// Severity of a change for users of the provider.
type Severity string

// Severities, most severe first.
const (
	Breaking Severity = "breaking" // existing configurations can fail or change behaviour
	Additive Severity = "additive" // new resources, attributes, values or filters
	Info     Severity = "info"     // visible but harmless (validation tweaks, sensitivity)
)

// Change is one difference in the provider surface.
type Change struct {
	Severity Severity `json:"severity"`
	// Type is the Terraform resource or data source type, e.g. netbox_site.
	Type string `json:"type"`
	// Attribute is the Terraform attribute path ("" for the type itself).
	Attribute string `json:"attribute,omitempty"`
	Detail    string `json:"detail"`
}

// Verdict is the machine-readable summary written by -json.
type Verdict struct {
	OldVersion string   `json:"old_version"`
	NewVersion string   `json:"new_version"`
	Breaking   int      `json:"breaking"`
	Additive   int      `json:"additive"`
	Info       int      `json:"info"`
	Changes    []Change `json:"changes"`
}

func main() {
	var (
		oldSpec = flag.String("old", "", "OpenAPI document of the pinned NetBox version")
		newSpec = flag.String("new", "", "OpenAPI document of the candidate NetBox version")
		ovDir   = flag.String("overrides", "generator/overrides", "directory of overrides YAML files")
		mdOut   = flag.String("markdown", "-", "write the Markdown report here (- for stdout, empty to skip)")
		jsonOut = flag.String("json", "", "write the JSON verdict here")
	)
	flag.Parse()
	if *oldSpec == "" || *newSpec == "" {
		fatal(errors.New("-old and -new are required"))
	}
	ov, err := overrides.Load(*ovDir)
	if err != nil {
		fatal(err)
	}
	oldRes, oldVer, err := load(*oldSpec, ov)
	if err != nil {
		fatal(err)
	}
	// A fresh copy, so the two builds cannot share mutable override state.
	ov, err = overrides.Load(*ovDir)
	if err != nil {
		fatal(err)
	}
	newRes, newVer, err := load(*newSpec, ov)
	if err != nil {
		fatal(err)
	}

	v := Verdict{OldVersion: oldVer, NewVersion: newVer, Changes: Diff(oldRes, newRes)}
	for _, c := range v.Changes {
		switch c.Severity {
		case Breaking:
			v.Breaking++
		case Additive:
			v.Additive++
		default:
			v.Info++
		}
	}
	if v.Changes == nil {
		v.Changes = []Change{}
	}

	if *jsonOut != "" {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fatal(err)
		}
		if err := os.WriteFile(*jsonOut, append(data, '\n'), 0o600); err != nil {
			fatal(err)
		}
	}
	switch *mdOut {
	case "":
	case "-":
		writeMarkdown(os.Stdout, v)
	default:
		f, err := os.Create(*mdOut)
		if err != nil {
			fatal(err)
		}
		writeMarkdown(f, v)
		if err := f.Close(); err != nil {
			fatal(err)
		}
	}
}

func load(path string, ov map[string]*overrides.Resource) ([]*model.Resource, string, error) {
	fixes := filepath.Join(filepath.Dir(path), "required-fixes.json")
	if _, err := os.Stat(fixes); err != nil {
		return nil, "", fmt.Errorf("%s: %w (copy spec/required-fixes.json next to the document)", path, err)
	}
	doc, err := openapi.Load(path)
	if err != nil {
		return nil, "", err
	}
	res, _, err := build.Build(doc, ov)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", path, err)
	}
	return res, doc.Info.Version, nil
}

// Diff compares two generator models. Skipped collections do not exist for
// users and are ignored; changes come out ordered by type, then attribute.
func Diff(oldRes, newRes []*model.Resource) []Change {
	oldBy, newBy := index(oldRes), index(newRes)
	var out []Change
	for _, name := range sortedKeys(oldBy, newBy) {
		o, n := oldBy[name], newBy[name]
		tf := "netbox_" + name
		switch {
		case o == nil:
			what := "new resource and data sources"
			if n.DataSourceOnly {
				what = "new data sources"
			}
			out = append(out, Change{Additive, tf, "", what})
			continue
		case n == nil:
			out = append(out, Change{Breaking, tf, "", "removed"})
			continue
		case !o.DataSourceOnly && n.DataSourceOnly:
			out = append(out, Change{Breaking, tf, "", "resource removed; only the data sources remain"})
			continue
		case o.DataSourceOnly && !n.DataSourceOnly:
			out = append(out, Change{Additive, tf, "", "new resource (previously data sources only)"})
		}
		out = append(out, diffAttrs(tf, "", o.Attrs, n.Attrs)...)
		out = append(out, diffDataSourceAttrs(tf, o.ReadOnlyAttrs, n.ReadOnlyAttrs)...)
		out = append(out, diffFilters("netbox_"+n.Plural, o.Filters, n.Filters)...)
	}
	return out
}

func index(rs []*model.Resource) map[string]*model.Resource {
	m := map[string]*model.Resource{}
	for _, r := range rs {
		if !r.Skip {
			m[r.Name] = r
		}
	}
	return m
}

func diffAttrs(tf, prefix string, oldAttrs, newAttrs []model.Attr) []Change {
	oldBy, newBy := attrIndex(oldAttrs), attrIndex(newAttrs)
	var out []Change
	for _, name := range sortedKeys(oldBy, newBy) {
		o, n := oldBy[name], newBy[name]
		path := prefix + name
		add := func(s Severity, format string, args ...any) {
			out = append(out, Change{s, tf, path, fmt.Sprintf(format, args...)})
		}
		switch {
		case o == nil:
			if n.Required && !n.ReadOnly {
				add(Breaking, "new required attribute")
			} else if n.ReadOnly {
				add(Additive, "new computed attribute")
			} else {
				add(Additive, "new optional attribute")
			}
			continue
		case n == nil:
			add(Breaking, "attribute removed")
			continue
		}
		if o.Kind != n.Kind {
			add(Breaking, "type changed from %s to %s", o.Kind, n.Kind)
			continue
		}
		switch {
		case !o.ReadOnly && n.ReadOnly:
			add(Breaking, "now read-only (computed only)")
		case o.ReadOnly && !n.ReadOnly:
			add(Additive, "now configurable")
		}
		switch {
		case !o.Required && n.Required && !n.ReadOnly:
			add(Breaking, "now required")
		case o.Required && !n.Required:
			add(Additive, "now optional")
		}
		switch {
		case !o.RequiresReplace && n.RequiresReplace:
			add(Breaking, "changes now force replacement")
		case o.RequiresReplace && !n.RequiresReplace:
			add(Additive, "changes no longer force replacement")
		}
		if added, removed := setDiff(o.Enum, n.Enum); len(added)+len(removed) > 0 {
			if len(removed) > 0 {
				add(Breaking, "values removed: %s", strings.Join(removed, ", "))
			}
			if len(added) > 0 {
				add(Additive, "values added: %s", strings.Join(added, ", "))
			}
		}
		if added, removed := setDiff(ints(o.EnumInts), ints(n.EnumInts)); len(added)+len(removed) > 0 {
			if len(removed) > 0 {
				add(Breaking, "values removed: %s", strings.Join(removed, ", "))
			}
			if len(added) > 0 {
				add(Additive, "values added: %s", strings.Join(added, ", "))
			}
		}
		if o.Sensitive != n.Sensitive {
			add(Info, "sensitive: %v -> %v", o.Sensitive, n.Sensitive)
		}
		if o.Pattern != n.Pattern || !eqIntPtr(o.MaxLength, n.MaxLength) || !eqIntPtr(o.MinLength, n.MinLength) {
			add(Info, "validation changed (length or pattern)")
		}
		if o.Kind == model.KindNestedList {
			out = append(out, diffAttrs(tf, path+".", o.Nested, n.Nested)...)
		}
	}
	return out
}

func diffDataSourceAttrs(tf string, oldAttrs, newAttrs []model.Attr) []Change {
	oldBy, newBy := attrIndex(oldAttrs), attrIndex(newAttrs)
	var out []Change
	for _, name := range sortedKeys(oldBy, newBy) {
		o, n := oldBy[name], newBy[name]
		switch {
		case o == nil:
			out = append(out, Change{Additive, tf, name, "new data source attribute"})
		case n == nil:
			out = append(out, Change{Breaking, tf, name, "data source attribute removed"})
		case o.Kind != n.Kind:
			out = append(out, Change{Breaking, tf, name, fmt.Sprintf("data source attribute type changed from %s to %s", o.Kind, n.Kind)})
		}
	}
	return out
}

// diffFilters compares list data source filters. Filter names are validated at
// plan time, so a removed filter fails configurations that use it.
func diffFilters(tf string, oldF, newF []model.Filter) []Change {
	names := func(fs []model.Filter) []string {
		s := make([]string, 0, len(fs))
		for _, f := range fs {
			s = append(s, f.Name)
		}
		return s
	}
	added, removed := setDiff(names(oldF), names(newF))
	var out []Change
	if len(removed) > 0 {
		out = append(out, Change{Breaking, tf, "filters", "filters removed: " + strings.Join(removed, ", ")})
	}
	if len(added) > 0 {
		out = append(out, Change{Additive, tf, "filters", "filters added: " + strings.Join(added, ", ")})
	}
	return out
}

func attrIndex(as []model.Attr) map[string]*model.Attr {
	m := map[string]*model.Attr{}
	for i := range as {
		m[as[i].Name] = &as[i]
	}
	return m
}

func sortedKeys[V any](a, b map[string]V) []string {
	seen := map[string]bool{}
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// setDiff returns the sorted values only in b (added) and only in a (removed).
func setDiff(a, b []string) (added, removed []string) {
	for _, v := range b {
		if !slices.Contains(a, v) {
			added = append(added, v)
		}
	}
	for _, v := range a {
		if !slices.Contains(b, v) {
			removed = append(removed, v)
		}
	}
	slices.Sort(added)
	slices.Sort(removed)
	return added, removed
}

func ints(vs []int64) []string {
	s := make([]string, 0, len(vs))
	for _, v := range vs {
		s = append(s, fmt.Sprint(v))
	}
	return s
}

func eqIntPtr(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func writeMarkdown(w io.Writer, v Verdict) {
	fmt.Fprintf(w, "### Provider surface: %s -> %s\n\n", v.OldVersion, v.NewVersion)
	if len(v.Changes) == 0 {
		fmt.Fprintln(w, "No change to any generated resource, data source, attribute or filter.")
		return
	}
	fmt.Fprintf(w, "%d breaking, %d additive, %d informational.\n\n", v.Breaking, v.Additive, v.Info)
	fmt.Fprintln(w, "| Severity | Type | Attribute | Change |")
	fmt.Fprintln(w, "|---|---|---|---|")
	for _, sev := range []Severity{Breaking, Additive, Info} {
		for _, c := range v.Changes {
			if c.Severity != sev {
				continue
			}
			attr := c.Attribute
			if attr != "" {
				attr = "`" + attr + "`"
			}
			fmt.Fprintf(w, "| %s | `%s` | %s | %s |\n", c.Severity, c.Type, attr, strings.ReplaceAll(c.Detail, "|", `\|`))
		}
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "netbox-diff:", err)
	os.Exit(1)
}
