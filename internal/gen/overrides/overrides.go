// Package overrides loads the hand-maintained per-app YAML files that tune the
// generator: naming, skips, classification hints, plan modifiers and test
// fixtures. See docs/GENERATOR.md for the schema.
package overrides

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// File is the top-level structure of generator/overrides/<app>.yaml.
type File struct {
	// Resources is keyed by API path segment ("sites", "ip-addresses") or,
	// when a segment exists in several apps, by "app/segment"
	// ("virtualization/interfaces").
	Resources map[string]*Resource `yaml:"resources"`
}

// Resource overrides one collection.
type Resource struct {
	Name               string                `yaml:"name"`
	Plural             string                `yaml:"plural"`
	Skip               bool                  `yaml:"skip"`
	SkipReason         string                `yaml:"skip_reason"`
	Description        string                `yaml:"description"`
	Lookups            []string              `yaml:"lookups"`
	ImportVerifyIgnore []string              `yaml:"import_verify_ignore"`
	SerialTest         bool                  `yaml:"serial_test"`
	Attributes         map[string]*Attribute `yaml:"attributes"`
	Test               *Test                 `yaml:"test"`
}

// Attribute overrides one API property (keyed by API property name).
type Attribute struct {
	Name            string `yaml:"name"`
	Kind            string `yaml:"kind"`
	Target          string `yaml:"target"`
	Skip            bool   `yaml:"skip"`
	RequiresReplace bool   `yaml:"requires_replace"`
	Sensitive       bool   `yaml:"sensitive"`
	Computed        *bool  `yaml:"computed"`
	// ReadOnly keeps the attribute as Computed only (never written), e.g. the
	// reverse side of a relation.
	ReadOnly bool `yaml:"read_only"`
	// Expose includes a read-only API property (id/url excluded) as a Computed
	// resource attribute; the prior state is kept when the API returns null
	// (write-once values such as token secrets).
	Expose      bool     `yaml:"expose"`
	Optional    *bool    `yaml:"optional"`
	Precision   int      `yaml:"precision"`
	OrderedList bool     `yaml:"ordered_list"`
	Description string   `yaml:"description"`
	Enum        []string `yaml:"enum"`
	// Nested overrides apply to the item attributes of nested lists.
	Nested map[string]*Attribute `yaml:"nested"`
}

// Test overrides the acceptance test.
type Test struct {
	Basic              string            `yaml:"basic"`
	Update             string            `yaml:"update"`
	Skip               string            `yaml:"skip"`
	ImportVerifyIgnore []string          `yaml:"import_verify_ignore"`
	Checks             map[string]string `yaml:"checks"`
}

// Load reads every *.yaml file in dir (sorted by name) and merges them. A
// resource may appear in several files (e.g. naming in one, fixtures in
// another): non-empty scalar fields from later files win, attribute maps are
// merged per property, and a later test block replaces an earlier one.
func Load(dir string) (map[string]*Resource, error) {
	out := map[string]*Resource{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && (filepath.Ext(e.Name()) == ".yaml" || filepath.Ext(e.Name()) == ".yml") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		data, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return nil, err
		}
		var f File
		if err := yaml.Unmarshal(data, &f); err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		for seg, r := range f.Resources {
			if r == nil {
				r = &Resource{}
			}
			if prev, dup := out[seg]; dup {
				merge(prev, r)
				continue
			}
			out[seg] = r
		}
	}
	return out, nil
}

func merge(dst, src *Resource) {
	if src.Name != "" {
		dst.Name = src.Name
	}
	if src.Plural != "" {
		dst.Plural = src.Plural
	}
	if src.Skip {
		dst.Skip = true
	}
	if src.SkipReason != "" {
		dst.SkipReason = src.SkipReason
	}
	if src.Description != "" {
		dst.Description = src.Description
	}
	if src.Lookups != nil {
		dst.Lookups = src.Lookups
	}
	dst.ImportVerifyIgnore = append(dst.ImportVerifyIgnore, src.ImportVerifyIgnore...)
	if src.SerialTest {
		dst.SerialTest = true
	}
	if len(src.Attributes) > 0 {
		if dst.Attributes == nil {
			dst.Attributes = map[string]*Attribute{}
		}
		for k, v := range src.Attributes {
			dst.Attributes[k] = v
		}
	}
	if src.Test != nil {
		dst.Test = src.Test
	}
}
