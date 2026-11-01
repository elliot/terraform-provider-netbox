package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
)

// The structs below mirror the parts of `terraform providers schema -json`
// (format_version 1.0) that matter for compatibility. Attribute descriptions
// are only counted, never classified.

type document struct {
	FormatVersion   string                     `json:"format_version"`
	ProviderSchemas map[string]*providerSchema `json:"provider_schemas"`
}

type providerSchema struct {
	Provider                 *schema                    `json:"provider"`
	ResourceSchemas          map[string]*schema         `json:"resource_schemas"`
	DataSourceSchemas        map[string]*schema         `json:"data_source_schemas"`
	EphemeralResourceSchemas map[string]*schema         `json:"ephemeral_resource_schemas"`
	ListResourceSchemas      map[string]*schema         `json:"list_resource_schemas"`
	ActionSchemas            map[string]*schema         `json:"action_schemas"`
	Functions                map[string]*function       `json:"functions"`
	ResourceIdentitySchemas  map[string]*identitySchema `json:"resource_identity_schemas"`
}

type schema struct {
	Version int64  `json:"version"`
	Block   *block `json:"block"`
}

type block struct {
	Attributes map[string]*attribute `json:"attributes"`
	BlockTypes map[string]*blockType `json:"block_types"`
	Deprecated bool                  `json:"deprecated"`
}

type attribute struct {
	Type        json.RawMessage `json:"type"`
	NestedType  *nestedType     `json:"nested_type"`
	Required    bool            `json:"required"`
	Optional    bool            `json:"optional"`
	Computed    bool            `json:"computed"`
	Sensitive   bool            `json:"sensitive"`
	Deprecated  bool            `json:"deprecated"`
	WriteOnly   bool            `json:"write_only"`
	Description string          `json:"description"`
}

type nestedType struct {
	Attributes  map[string]*attribute `json:"attributes"`
	NestingMode string                `json:"nesting_mode"`
	MinItems    uint64                `json:"min_items"`
	MaxItems    uint64                `json:"max_items"`
}

type blockType struct {
	NestingMode string `json:"nesting_mode"`
	Block       *block `json:"block"`
	MinItems    uint64 `json:"min_items"`
	MaxItems    uint64 `json:"max_items"`
}

type identitySchema struct {
	Version    int64                         `json:"version"`
	Attributes map[string]*identityAttribute `json:"attributes"`
}

type identityAttribute struct {
	Type              json.RawMessage `json:"type"`
	RequiredForImport bool            `json:"required_for_import"`
	OptionalForImport bool            `json:"optional_for_import"`
}

type function struct {
	ReturnType         json.RawMessage `json:"return_type"`
	Parameters         []*parameter    `json:"parameters"`
	VariadicParameter  *parameter      `json:"variadic_parameter"`
	DeprecationMessage string          `json:"deprecation_message"`
}

type parameter struct {
	Name       string          `json:"name"`
	Type       json.RawMessage `json:"type"`
	IsNullable bool            `json:"is_nullable"`
}

// loadProvider reads a schema document and returns the schema of one provider.
// An empty address selects the only provider in the document.
func loadProvider(path, address string) (*providerSchema, string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from the -old/-new flags
	if err != nil {
		return nil, "", err
	}
	return parseProvider(data, address)
}

func parseProvider(data []byte, address string) (*providerSchema, string, error) {
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, "", fmt.Errorf("not a `terraform providers schema -json` document: %w", err)
	}
	if doc.FormatVersion == "" || !strings.HasPrefix(doc.FormatVersion, "1.") {
		return nil, "", fmt.Errorf("unsupported format_version %q (want 1.x)", doc.FormatVersion)
	}
	if address != "" {
		p, ok := doc.ProviderSchemas[address]
		if !ok {
			return nil, "", fmt.Errorf("provider %s not in document (have %s)", address, strings.Join(sortedKeys(doc.ProviderSchemas), ", "))
		}
		return p, address, nil
	}
	if len(doc.ProviderSchemas) != 1 {
		return nil, "", fmt.Errorf("document has %d providers (%s); pick one with -provider", len(doc.ProviderSchemas), strings.Join(sortedKeys(doc.ProviderSchemas), ", "))
	}
	for addr, p := range doc.ProviderSchemas {
		return p, addr, nil
	}
	panic("unreachable")
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// unionKeys returns the sorted keys present in either map.
func unionKeys[V any](a, b map[string]V) []string {
	keys := sortedKeys(a)
	for k := range b {
		if _, ok := a[k]; !ok {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}

// canonicalType re-encodes a JSON type expression so equal types compare equal
// regardless of whitespace or object key order.
func canonicalType(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(out)
}

// typeString renders a JSON type expression in Terraform's type syntax,
// e.g. ["set","number"] -> set(number).
func typeString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "(none)"
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return renderType(v)
}

func renderType(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		if len(t) < 2 {
			break
		}
		kind, _ := t[0].(string)
		switch kind {
		case "list", "set", "map":
			return kind + "(" + renderType(t[1]) + ")"
		case "object":
			attrs, _ := t[1].(map[string]any)
			parts := make([]string, 0, len(attrs))
			for _, k := range sortedKeys(attrs) {
				parts = append(parts, k+"="+renderType(attrs[k]))
			}
			return "object({" + strings.Join(parts, ", ") + "})"
		case "tuple":
			elems, _ := t[1].([]any)
			parts := make([]string, 0, len(elems))
			for _, e := range elems {
				parts = append(parts, renderType(e))
			}
			return "tuple([" + strings.Join(parts, ", ") + "])"
		}
	}
	out, _ := json.Marshal(v)
	return string(out)
}

// impliedType is the type Terraform derives for an attribute: its declared
// type, or for a nested attribute the collection of objects it describes.
func impliedType(a *attribute) string {
	if a.NestedType == nil {
		return canonicalType(a.Type)
	}
	attrs := map[string]json.RawMessage{}
	for name, child := range a.NestedType.Attributes {
		attrs[name] = json.RawMessage(impliedType(child))
	}
	obj, _ := json.Marshal([]any{"object", attrs})
	switch a.NestedType.NestingMode {
	case "list", "set", "map":
		out, _ := json.Marshal([]any{a.NestedType.NestingMode, json.RawMessage(obj)})
		return string(out)
	default: // single
		return string(obj)
	}
}

// describeType is the human-readable form used in the report.
func describeType(a *attribute) string {
	if a.NestedType == nil {
		return typeString(a.Type)
	}
	return "nested " + a.NestedType.NestingMode
}
