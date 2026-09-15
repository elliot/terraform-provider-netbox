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
	Name            string   `yaml:"name"`
	Kind            string   `yaml:"kind"`
	Target          string   `yaml:"target"`
	Skip            bool     `yaml:"skip"`
	RequiresReplace bool     `yaml:"requires_replace"`
	Sensitive       bool     `yaml:"sensitive"`
	Computed        *bool    `yaml:"computed"`
	Optional        *bool    `yaml:"optional"`
	Precision       int      `yaml:"precision"`
	OrderedList     bool     `yaml:"ordered_list"`
	Description     string   `yaml:"description"`
	Enum            []string `yaml:"enum"`
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

// Load reads every *.yaml file in dir and merges them; later files must not
// redefine a path segment already defined.
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
			if _, dup := out[seg]; dup {
				return nil, fmt.Errorf("%s: resource %q already defined in another overrides file", n, seg)
			}
			if r == nil {
				r = &Resource{}
			}
			out[seg] = r
		}
	}
	return out, nil
}
