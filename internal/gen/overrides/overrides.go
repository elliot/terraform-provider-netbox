// Package overrides loads the hand-maintained per-app YAML files that tune the
// generator: naming, skips, classification hints, plan modifiers and test
// fixtures. See docs/GENERATOR.md for the schema.
package overrides

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	Name               string   `yaml:"name"`
	Plural             string   `yaml:"plural"`
	Skip               bool     `yaml:"skip"`
	SkipReason         string   `yaml:"skip_reason"`
	Description        string   `yaml:"description"`
	Lookups            []string `yaml:"lookups"`
	ImportVerifyIgnore []string `yaml:"import_verify_ignore"`
	SerialTest         bool     `yaml:"serial_test"`
	// Deprecated marks the resource and both data sources as deprecated.
	Deprecated Deprecation `yaml:"deprecated"`
	// SchemaVersion is the resource schema version. Above 0 the resource
	// implements ResourceWithUpgradeState with the upgraders registered in
	// internal/upgrade.
	SchemaVersion int64                 `yaml:"schema_version"`
	Attributes    map[string]*Attribute `yaml:"attributes"`
	Test          *Test                 `yaml:"test"`
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
	Expose   bool  `yaml:"expose"`
	Optional *bool `yaml:"optional"`
	// Default makes an integer or FK attribute Optional+Computed with a static
	// default value (a required property becomes optional).
	Default   *int64 `yaml:"default"`
	Precision int    `yaml:"precision"`
	// Format names a value format NetBox normalises (mac, wwn): the attribute
	// gets a format validator and keeps the configured letter case on read.
	Format      string   `yaml:"format"`
	OrderedList bool     `yaml:"ordered_list"`
	Description string   `yaml:"description"`
	Enum        []string `yaml:"enum"`
	// Deprecated sets the attribute's DeprecationMessage. Properties the spec
	// marks deprecated get a default message unless this is false.
	Deprecated Deprecation `yaml:"deprecated"`
	// Nested overrides apply to the item attributes of nested lists.
	Nested map[string]*Attribute `yaml:"nested"`
}

// Deprecation is the value of a deprecated key: a message, true (use the
// default message) or false (suppress the deprecation the spec declares).
type Deprecation struct {
	// Set is true when the key is present.
	Set bool
	// Off is true for `deprecated: false`.
	Off bool
	// Message is the configured message ("" for `deprecated: true`).
	Message string
}

// UnmarshalYAML accepts a string or a boolean.
func (d *Deprecation) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("line %d: deprecated must be a message or a boolean", n.Line)
	}
	*d = Deprecation{Set: true}
	if n.Tag == "!!bool" {
		var b bool
		if err := n.Decode(&b); err != nil {
			return err
		}
		d.Off = !b
		return nil
	}
	d.Message = strings.TrimSpace(n.Value)
	if d.Message == "" {
		return fmt.Errorf("line %d: deprecated message is empty", n.Line)
	}
	return nil
}

// Resolve returns the deprecation message for a property: specDeprecated is
// whether the OpenAPI document marks it deprecated, def the message used when
// no override gives one. It returns "" when the property is not deprecated.
func (d Deprecation) Resolve(specDeprecated bool, def string) string {
	switch {
	case d.Off:
		return ""
	case d.Message != "":
		return d.Message
	case d.Set || specDeprecated:
		return def
	}
	return ""
}

// Test overrides the acceptance test.
type Test struct {
	Basic              string            `yaml:"basic"`
	Update             string            `yaml:"update"`
	Skip               string            `yaml:"skip"`
	ImportVerifyIgnore []string          `yaml:"import_verify_ignore"`
	Checks             map[string]string `yaml:"checks"`
	Upgrade            *UpgradeTest      `yaml:"upgrade"`
}

// UpgradeTest overrides the upgrade test (TestAcc<Resource>_upgrade).
type UpgradeTest struct {
	// Skip disables the upgrade test with a reason.
	Skip string `yaml:"skip"`
	// Config is complete HCL the released provider accepts; it replaces the
	// default of trying test.update, then test.basic.
	Config string `yaml:"config"`
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
	if src.Deprecated.Set {
		dst.Deprecated = src.Deprecated
	}
	if src.SchemaVersion != 0 {
		dst.SchemaVersion = src.SchemaVersion
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
