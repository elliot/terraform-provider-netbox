// Package openapi is a minimal OpenAPI 3.0 reader tailored to the NetBox
// schema: just enough structure to discover CRUD collections and classify
// request/response properties.
package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Document is the parsed OpenAPI document.
type Document struct {
	Info       Info                 `json:"info"`
	Paths      map[string]*PathItem `json:"paths"`
	Components Components           `json:"components"`
}

// Info holds the document metadata.
type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

// Components holds reusable schemas.
type Components struct {
	Schemas map[string]*Schema `json:"schemas"`
}

// PathItem groups the operations of one path.
type PathItem struct {
	Get    *Operation `json:"get"`
	Post   *Operation `json:"post"`
	Put    *Operation `json:"put"`
	Patch  *Operation `json:"patch"`
	Delete *Operation `json:"delete"`
}

// Operation is one HTTP operation.
type Operation struct {
	OperationID string               `json:"operationId"`
	Description string               `json:"description"`
	Tags        []string             `json:"tags"`
	Parameters  []*Parameter         `json:"parameters"`
	RequestBody *RequestBody         `json:"requestBody"`
	Responses   map[string]*Response `json:"responses"`
}

// Parameter is a query or path parameter.
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Description string  `json:"description"`
	Required    bool    `json:"required"`
	Schema      *Schema `json:"schema"`
	Explode     *bool   `json:"explode"`
}

// RequestBody is an operation request body.
type RequestBody struct {
	Required bool                  `json:"required"`
	Content  map[string]*MediaType `json:"content"`
}

// Response is an operation response.
type Response struct {
	Description string                `json:"description"`
	Content     map[string]*MediaType `json:"content"`
}

// MediaType wraps a schema for one content type.
type MediaType struct {
	Schema *Schema `json:"schema"`
}

// Schema is an OpenAPI schema object (subset).
type Schema struct {
	Ref                  string             `json:"$ref"`
	Type                 string             `json:"type"`
	Format               string             `json:"format"`
	Description          string             `json:"description"`
	Title                string             `json:"title"`
	Nullable             bool               `json:"nullable"`
	ReadOnly             bool               `json:"readOnly"`
	WriteOnly            bool               `json:"writeOnly"`
	Deprecated           bool               `json:"deprecated"`
	Enum                 []any              `json:"enum"`
	Default              any                `json:"default"`
	MaxLength            *int               `json:"maxLength"`
	MinLength            *int               `json:"minLength"`
	Pattern              string             `json:"pattern"`
	Minimum              *float64           `json:"minimum"`
	Maximum              *float64           `json:"maximum"`
	MinItems             *int               `json:"minItems"`
	MaxItems             *int               `json:"maxItems"`
	Items                *Schema            `json:"items"`
	Properties           map[string]*Schema `json:"properties"`
	PropertyOrder        []string           `json:"-"`
	Required             []string           `json:"required"`
	AllOf                []*Schema          `json:"allOf"`
	OneOf                []*Schema          `json:"oneOf"`
	AnyOf                []*Schema          `json:"anyOf"`
	AdditionalProperties json.RawMessage    `json:"additionalProperties"`
	SpecEnumID           string             `json:"x-spec-enum-id"`
}

// UnmarshalJSON keeps property declaration order, which openapi-generator uses
// for constructor argument order.
func (s *Schema) UnmarshalJSON(data []byte) error {
	type alias Schema
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = Schema(a)
	s.PropertyOrder = orderedKeys(data, "properties")
	return nil
}

// orderedKeys returns the keys of the object under field in declaration order.
func orderedKeys(data []byte, field string) []string {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	// Walk the top-level object.
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil
		}
		key, _ := keyTok.(string)
		if key != field {
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return nil
			}
			continue
		}
		// Now positioned at the properties object.
		tok, err := dec.Token()
		if err != nil || tok != json.Delim('{') {
			return nil
		}
		var keys []string
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return nil
			}
			k, _ := kt.(string)
			keys = append(keys, k)
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return nil
			}
		}
		return keys
	}
	return nil
}

// Load reads and parses an OpenAPI JSON document and applies the required-list
// corrections from required-fixes.json next to it (the same file the client
// generator uses), so both sides agree on constructor arguments.
func Load(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := doc.applyRequiredFixes(filepath.Join(filepath.Dir(path), "required-fixes.json")); err != nil {
		return nil, err
	}
	return &doc, nil
}

type requiredFix struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

func (d *Document) applyRequiredFixes(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	for name, msg := range raw {
		if strings.HasPrefix(name, "_") {
			continue
		}
		var fix requiredFix
		if err := json.Unmarshal(msg, &fix); err != nil {
			return fmt.Errorf("parse %s: %s: %w", path, name, err)
		}
		s := d.Components.Schemas[name]
		if s == nil {
			continue
		}
		var req []string
		for _, r := range s.Required {
			keep := true
			for _, rm := range fix.Remove {
				if rm == r {
					keep = false
				}
			}
			if keep {
				req = append(req, r)
			}
		}
		for _, a := range fix.Add {
			if !contains(req, a) {
				req = append(req, a)
			}
		}
		s.Required = req
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// RefName returns the component name of a $ref ("#/components/schemas/Site" -> "Site").
func RefName(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

// Resolve follows $ref (and single-element allOf wrappers) to the named
// component schema. It returns the schema and the component name ("" when the
// schema is inline).
func (d *Document) Resolve(s *Schema) (*Schema, string) {
	name := ""
	for i := 0; i < 8 && s != nil; i++ {
		switch {
		case s.Ref != "":
			name = RefName(s.Ref)
			s = d.Components.Schemas[name]
		case len(s.AllOf) == 1 && s.Ref == "" && len(s.Properties) == 0:
			// {allOf: [{$ref}], nullable: true}
			inner := s.AllOf[0]
			if inner.Ref != "" {
				name = RefName(inner.Ref)
				s = d.Components.Schemas[name]
			} else {
				s = inner
			}
		default:
			return s, name
		}
	}
	return s, name
}

// Operations returns the operations of a path item keyed by lowercase method.
func (p *PathItem) Operations() map[string]*Operation {
	out := map[string]*Operation{}
	if p.Get != nil {
		out["get"] = p.Get
	}
	if p.Post != nil {
		out["post"] = p.Post
	}
	if p.Put != nil {
		out["put"] = p.Put
	}
	if p.Patch != nil {
		out["patch"] = p.Patch
	}
	if p.Delete != nil {
		out["delete"] = p.Delete
	}
	return out
}

// BodySchema returns the JSON request body schema of an operation, unwrapping
// NetBox's bulk “oneOf [X, X[]]“ form to X.
func (op *Operation) BodySchema() *Schema {
	if op == nil || op.RequestBody == nil {
		return nil
	}
	mt := op.RequestBody.Content["application/json"]
	if mt == nil {
		for _, m := range op.RequestBody.Content {
			mt = m
			break
		}
	}
	if mt == nil || mt.Schema == nil {
		return nil
	}
	s := mt.Schema
	if len(s.OneOf) > 0 {
		for _, alt := range s.OneOf {
			if alt.Type != "array" {
				return alt
			}
		}
	}
	return s
}

// ResponseSchema returns the JSON schema of the first 2xx response.
func (op *Operation) ResponseSchema() *Schema {
	if op == nil {
		return nil
	}
	codes := make([]string, 0, len(op.Responses))
	for c := range op.Responses {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	for _, c := range codes {
		if !strings.HasPrefix(c, "2") {
			continue
		}
		r := op.Responses[c]
		if r == nil {
			continue
		}
		if mt := r.Content["application/json"]; mt != nil {
			return mt.Schema
		}
	}
	return nil
}

// EnumStrings returns the string members of an enum, dropping null and "".
func (s *Schema) EnumStrings() []string {
	var out []string
	for _, e := range s.Enum {
		if str, ok := e.(string); ok && str != "" {
			out = append(out, str)
		}
	}
	return out
}

// EnumInts returns the integer members of an enum.
func (s *Schema) EnumInts() []int64 {
	var out []int64
	for _, e := range s.Enum {
		if f, ok := e.(float64); ok {
			out = append(out, int64(f))
		}
	}
	return out
}

// HasAdditionalProperties reports whether the schema is a free-form object.
func (s *Schema) HasAdditionalProperties() bool {
	return len(s.AdditionalProperties) > 0 && string(s.AdditionalProperties) != "false"
}

// IsRequired reports whether prop is in the schema's required list.
func (s *Schema) IsRequired(prop string) bool {
	for _, r := range s.Required {
		if r == prop {
			return true
		}
	}
	return false
}

// OrderedProperties returns property names in declaration order.
func (s *Schema) OrderedProperties() []string {
	if len(s.PropertyOrder) == len(s.Properties) {
		return s.PropertyOrder
	}
	keys := make([]string, 0, len(s.Properties))
	for k := range s.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
