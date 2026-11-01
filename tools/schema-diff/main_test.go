package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/report.md")

func TestFixtures(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-old", "testdata/old.json", "-new", "testdata/new.json", "-old-label", "v1.0.0", "-new-label", "HEAD"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code %d, want 1 (stderr: %s)", code, stderr.String())
	}
	golden := filepath.Join("testdata", "report.md")
	if *update {
		if err := os.WriteFile(golden, stdout.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != string(want) {
		t.Errorf("report differs from %s (rerun with -update):\n%s", golden, got)
	}

	// The same input passes with -allow-breaking and still prints the report.
	stdout.Reset()
	if code := run([]string{"-old", "testdata/old.json", "-new", "testdata/new.json", "-allow-breaking"}, &stdout, &stderr); code != 0 {
		t.Fatalf("-allow-breaking: exit code %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Breaking changes found and allowed.") || !strings.Contains(stdout.String(), "### Breaking changes") {
		t.Errorf("-allow-breaking report:\n%s", stdout.String())
	}

	// Identical documents.
	stdout.Reset()
	if code := run([]string{"-old", "testdata/old.json", "-new", "testdata/old.json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("identical: exit code %d", code)
	}
	if !strings.Contains(stdout.String(), "No schema changes.") {
		t.Errorf("identical report:\n%s", stdout.String())
	}
}

func TestUsageErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	for _, args := range [][]string{
		{},
		{"-old", "testdata/old.json"},
		{"-old", "testdata/missing.json", "-new", "testdata/new.json"},
		{"-old", "testdata/old.json", "-new", "testdata/new.json", "-provider", "registry.terraform.io/other/x"},
	} {
		if code := run(args, &stdout, &stderr); code != 2 {
			t.Errorf("%v: exit code %d, want 2", args, code)
		}
	}
}

// diffJSON wraps provider schema JSON fragments into documents and diffs them.
func diffJSON(t *testing.T, before, after string) []Change {
	t.Helper()
	wrap := func(s string) []byte {
		return []byte(`{"format_version":"1.0","provider_schemas":{"registry.terraform.io/x/y":` + s + `}}`)
	}
	o, _, err := parseProvider(wrap(before), "")
	if err != nil {
		t.Fatal(err)
	}
	n, _, err := parseProvider(wrap(after), "")
	if err != nil {
		t.Fatal(err)
	}
	return Diff(o, n).Changes
}

// res builds a provider schema with one resource x_r holding the given attributes JSON.
func res(attrs string) string {
	return `{"resource_schemas":{"x_r":{"version":0,"block":{"attributes":{` + attrs + `}}}}}`
}

func TestAttributeChanges(t *testing.T) {
	tests := []struct {
		name, before, after string
		want                []Severity
		detail              string
	}{
		{"description only", `"a":{"type":"string","optional":true,"description":"x"}`, `"a":{"type":"string","optional":true,"description":"y"}`, nil, ""},
		{"required to optional", `"a":{"type":"string","required":true}`, `"a":{"type":"string","optional":true}`, []Severity{Additive}, "required -> optional"},
		{"required to optional+computed", `"a":{"type":"string","required":true}`, `"a":{"type":"string","optional":true,"computed":true}`, []Severity{Additive}, ""},
		{"required to computed", `"a":{"type":"string","required":true}`, `"a":{"type":"string","computed":true}`, []Severity{Breaking}, "computed-only"},
		{"optional to required", `"a":{"type":"string","optional":true}`, `"a":{"type":"string","required":true}`, []Severity{Breaking}, "omit it fail"},
		{"optional to optional+computed", `"a":{"type":"string","optional":true}`, `"a":{"type":"string","optional":true,"computed":true}`, []Severity{Notice}, ""},
		{"optional to computed", `"a":{"type":"string","optional":true}`, `"a":{"type":"string","computed":true}`, []Severity{Breaking}, "set it fail"},
		{"optional+computed to optional", `"a":{"type":"string","optional":true,"computed":true}`, `"a":{"type":"string","optional":true}`, []Severity{Breaking}, "plan a change"},
		{"optional+computed to required", `"a":{"type":"string","optional":true,"computed":true}`, `"a":{"type":"string","required":true}`, []Severity{Breaking}, ""},
		{"computed to optional+computed", `"a":{"type":"string","computed":true}`, `"a":{"type":"string","optional":true,"computed":true}`, []Severity{Additive}, ""},
		{"computed to optional", `"a":{"type":"string","computed":true}`, `"a":{"type":"string","optional":true}`, []Severity{Breaking}, ""},
		{"computed to required", `"a":{"type":"string","computed":true}`, `"a":{"type":"string","required":true}`, []Severity{Breaking}, ""},
		{"list to set", `"a":{"type":["list","string"],"optional":true}`, `"a":{"type":["set","string"],"optional":true}`, []Severity{Breaking}, "list(string) to set(string)"},
		{"object key order is irrelevant", `"a":{"type":["object",{"x":"string","y":"number"}],"optional":true}`, `"a":{"type":["object",{"y":"number","x":"string"}],"optional":true}`, nil, ""},
		{"sensitive added", `"a":{"type":"string","optional":true}`, `"a":{"type":"string","optional":true,"sensitive":true}`, []Severity{Notice}, "sensitive"},
		{"deprecated", `"a":{"type":"string","optional":true}`, `"a":{"type":"string","optional":true,"deprecated":true}`, []Severity{Notice}, "deprecated"},
		{"write-only", `"a":{"type":"string","optional":true}`, `"a":{"type":"string","optional":true,"write_only":true}`, []Severity{Breaking}, "write-only"},
		{"removed", `"a":{"type":"string","computed":true}`, ``, []Severity{Breaking}, "removed"},
		{"new required", ``, `"a":{"type":"string","required":true}`, []Severity{Breaking}, "new required"},
		{"new computed", ``, `"a":{"type":"string","computed":true}`, []Severity{Additive}, "new computed"},
		{
			"nested mode list to set",
			`"a":{"nested_type":{"nesting_mode":"list","attributes":{"x":{"type":"string","optional":true}}},"optional":true}`,
			`"a":{"nested_type":{"nesting_mode":"set","attributes":{"x":{"type":"string","optional":true}}},"optional":true}`,
			[]Severity{Breaking}, "nested list to nested set",
		},
		{
			"nested child type",
			`"a":{"nested_type":{"nesting_mode":"list","attributes":{"x":{"type":"string","optional":true}}},"optional":true}`,
			`"a":{"nested_type":{"nesting_mode":"list","attributes":{"x":{"type":"number","optional":true}}},"optional":true}`,
			[]Severity{Breaking}, "type changed",
		},
		{
			"nested child removed",
			`"a":{"nested_type":{"nesting_mode":"list","attributes":{"x":{"type":"string","optional":true},"y":{"type":"string","optional":true}}},"optional":true}`,
			`"a":{"nested_type":{"nesting_mode":"list","attributes":{"x":{"type":"string","optional":true}}},"optional":true}`,
			[]Severity{Breaking}, "attribute removed",
		},
		{
			"nested child becomes required",
			`"a":{"nested_type":{"nesting_mode":"list","attributes":{"x":{"type":"string","optional":true}}},"optional":true}`,
			`"a":{"nested_type":{"nesting_mode":"list","attributes":{"x":{"type":"string","required":true}}},"optional":true}`,
			[]Severity{Breaking}, "omit it fail",
		},
		{
			"flags below a computed-only nested attribute are ignored",
			`"a":{"nested_type":{"nesting_mode":"list","attributes":{"x":{"type":"string","computed":true}}},"computed":true}`,
			`"a":{"nested_type":{"nesting_mode":"list","attributes":{"x":{"type":"string","optional":true}}},"computed":true}`,
			nil, "",
		},
		{
			"object type to nested attribute with the same value type",
			`"a":{"type":["list",["object",{"x":"string"}]],"optional":true}`,
			`"a":{"nested_type":{"nesting_mode":"list","attributes":{"x":{"type":"string","optional":true}}},"optional":true}`,
			[]Severity{Notice}, "same value type",
		},
		{
			"nested max items tightened",
			`"a":{"nested_type":{"nesting_mode":"list","attributes":{"x":{"type":"string","optional":true}}},"optional":true}`,
			`"a":{"nested_type":{"nesting_mode":"list","max_items":2,"attributes":{"x":{"type":"string","optional":true}}},"optional":true}`,
			[]Severity{Breaking}, "at most 2 items (was unlimited)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffJSON(t, res(tt.before), res(tt.after))
			if len(got) != len(tt.want) {
				t.Fatalf("got %d changes %+v, want severities %v", len(got), got, tt.want)
			}
			for i, c := range got {
				if c.Severity != tt.want[i] {
					t.Errorf("change %d: %s (%s), want %s", i, c.Severity, c.Detail, tt.want[i])
				}
			}
			if tt.detail != "" && !strings.Contains(got[0].Detail, tt.detail) {
				t.Errorf("detail %q does not contain %q", got[0].Detail, tt.detail)
			}
		})
	}
}

func TestBlocksFunctionsIdentities(t *testing.T) {
	blk := func(mode string, minItems int, attrs string) string {
		return `{"resource_schemas":{"x_r":{"version":0,"block":{"block_types":{"b":{"nesting_mode":"` + mode +
			`","min_items":` + string(rune('0'+minItems)) + `,"block":{"attributes":{` + attrs + `}}}}}}}}`
	}
	fn := func(ret, params string) string {
		return `{"functions":{"f":{"return_type":` + ret + `,"parameters":[` + params + `]}}}`
	}
	ident := func(attrs string) string {
		return `{"resource_schemas":{"x_r":{"version":0,"block":{}}},"resource_identity_schemas":{"x_r":{"version":0,"attributes":{` + attrs + `}}}}`
	}
	tests := []struct {
		name, before, after string
		want                []Severity
	}{
		{"block removed", blk("list", 0, ""), res(""), []Severity{Breaking}},
		{"optional block added", res(""), blk("list", 0, ""), []Severity{Additive}},
		{"required block added", res(""), blk("list", 1, ""), []Severity{Breaking}},
		{"block mode changed", blk("list", 0, ""), blk("set", 0, ""), []Severity{Breaking}},
		{"block min items raised", blk("list", 0, ""), blk("list", 1, ""), []Severity{Breaking}},
		{"block attribute removed", blk("list", 0, `"x":{"type":"string","optional":true}`), blk("list", 0, ""), []Severity{Breaking}},
		{"resource removed", res(""), `{}`, []Severity{Breaking}},
		{"resource added", `{}`, res(""), []Severity{Additive}},
		{"data source removed", `{"data_source_schemas":{"x_d":{"version":0,"block":{}}}}`, `{}`, []Severity{Breaking}},
		{"ephemeral removed", `{"ephemeral_resource_schemas":{"x_e":{"version":0,"block":{}}}}`, `{}`, []Severity{Breaking}},
		{"schema version decreased", `{"resource_schemas":{"x_r":{"version":2,"block":{}}}}`, res(""), []Severity{Breaking}},
		{"function unchanged", fn(`"string"`, `{"name":"a","type":"string"}`), fn(`"string"`, `{"name":"b","type":"string"}`), nil},
		{"function return type", fn(`"string"`, ``), fn(`"number"`, ``), []Severity{Breaking}},
		{"function parameter added", fn(`"string"`, ``), fn(`"string"`, `{"name":"a","type":"string"}`), []Severity{Breaking}},
		{"function parameter type", fn(`"string"`, `{"name":"a","type":"string"}`), fn(`"string"`, `{"name":"a","type":"number"}`), []Severity{Breaking}},
		{"function removed", fn(`"string"`, ``), `{}`, []Severity{Breaking}},
		{"identity attribute removed", ident(`"id":{"type":"number","required_for_import":true}`), ident(``), []Severity{Breaking}},
		{"identity attribute required", ident(`"id":{"type":"number","optional_for_import":true}`), ident(`"id":{"type":"number","required_for_import":true}`), []Severity{Breaking}},
		{"identity attribute added optional", ident(``), ident(`"id":{"type":"number","optional_for_import":true}`), []Severity{Additive}},
		{"identity added", res(""), ident(`"id":{"type":"number","required_for_import":true}`), []Severity{Additive}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffJSON(t, tt.before, tt.after)
			if len(got) != len(tt.want) {
				t.Fatalf("got %+v, want severities %v", got, tt.want)
			}
			for i, c := range got {
				if c.Severity != tt.want[i] {
					t.Errorf("change %d: %s (%s), want %s", i, c.Severity, c.Detail, tt.want[i])
				}
			}
		})
	}
}

func TestReportFoldsLongSections(t *testing.T) {
	var changes []Change
	for i := range collapseAfter + 1 {
		changes = append(changes, Change{Additive, KindResource, "x_" + string(rune('a'+i)), "", "new resource"})
	}
	r := Report(Result{Changes: changes}, "a", "b", false)
	if !strings.Contains(r, "<details><summary>16 additive changes</summary>") || !strings.Contains(r, "No breaking changes.") {
		t.Errorf("report:\n%s", r)
	}
}

func TestLatestVersion(t *testing.T) {
	got, err := latestVersion([]byte(`{"versions":[{"version":"0.5.0"},{"version":"0.10.0"},{"version":"1.0.0-rc1"},{"version":"0.9.3"}]}`))
	if err != nil || got != "0.10.0" {
		t.Errorf("latestVersion = %q, %v", got, err)
	}
	if _, err := latestVersion([]byte(`{"versions":[]}`)); err == nil {
		t.Error("empty list: want error")
	}
}

func TestTypeString(t *testing.T) {
	for raw, want := range map[string]string{
		`"string"`:                               "string",
		`["set","number"]`:                       "set(number)",
		`["map",["list","bool"]]`:                "map(list(bool))",
		`["object",{"b":"number","a":"string"}]`: "object({a=string, b=number})",
		`["tuple",["string","number"]]`:          "tuple([string, number])",
	} {
		if got := typeString([]byte(raw)); got != want {
			t.Errorf("typeString(%s) = %s, want %s", raw, got, want)
		}
	}
}
