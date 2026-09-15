// Package build turns the NetBox OpenAPI document plus overrides into the
// generator's Resource model.
package build

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/elliot/terraform-provider-netbox/internal/gen/model"
	"github.com/elliot/terraform-provider-netbox/internal/gen/naming"
	"github.com/elliot/terraform-provider-netbox/internal/gen/openapi"
	"github.com/elliot/terraform-provider-netbox/internal/gen/overrides"
)

var collectionRe = regexp.MustCompile(`^/api/([a-z-]+)/([a-z0-9-]+)/$`)

// Skipped query parameters that are not object filters.
var nonFilterParams = map[string]bool{
	"limit": true, "offset": true, "ordering": true, "brief": true, "fields": true, "omit": true, "start": true,
}

// Builder holds state shared while building all resources.
type Builder struct {
	Doc       *openapi.Document
	Overrides map[string]*overrides.Resource
	// readSchemaToName maps a read component ("Site") to the Terraform name.
	readSchemaToName map[string]string
	Warnings         []string
}

// Build discovers every CRUD collection and returns the resources sorted by app and name.
func Build(doc *openapi.Document, ov map[string]*overrides.Resource) ([]*model.Resource, []string, error) {
	b := &Builder{Doc: doc, Overrides: ov, readSchemaToName: map[string]string{}}
	var paths []string
	for p := range doc.Paths {
		if collectionRe.MatchString(p) {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)

	// Pass 1: discover collections and names (needed for FK targets).
	type disc struct {
		path, app, seg string
		item           *openapi.PathItem
		detail         *openapi.PathItem
		crud           bool
	}
	var found []disc
	for _, p := range paths {
		m := collectionRe.FindStringSubmatch(p)
		item := doc.Paths[p]
		detail := doc.Paths[p+"{id}/"]
		crud := item.Post != nil && item.Get != nil && detail != nil && detail.Get != nil && detail.Patch != nil && detail.Delete != nil
		readOnly := item.Get != nil && item.Post == nil && detail != nil && detail.Get != nil && detail.Patch == nil
		if !crud && !readOnly {
			continue
		}
		if !strings.HasSuffix(item.Get.OperationID, "_list") {
			continue // e.g. core/background-tasks: not a standard list endpoint
		}
		found = append(found, disc{p, m[1], m[2], item, detail, crud})
	}
	seenNames := map[string]string{}
	seenPlural := map[string]string{}
	var resources []*model.Resource
	for _, d := range found {
		o := lookupOverride(ov, d.app, d.seg)
		r := &model.Resource{App: d.app, PathSegment: d.seg, Path: d.path, DataSourceOnly: !d.crud}
		snake := naming.Snake(d.seg)
		r.Plural = snake
		r.Name = naming.Singular(snake)
		if o.Name != "" {
			r.Name = o.Name
		}
		if o.Plural != "" {
			r.Plural = o.Plural
		}
		r.GoName = naming.GoIdent(r.Name)
		r.Skip = o.Skip
		r.SkipReason = o.SkipReason
		if prev, dup := seenNames[r.Name]; dup && !r.Skip {
			return nil, nil, fmt.Errorf("resource name collision %q between %s and %s: add a name override", r.Name, prev, d.path)
		}
		if prev, dup := seenPlural[r.Plural]; dup && !r.Skip {
			return nil, nil, fmt.Errorf("plural name collision %q between %s and %s: add a plural override", r.Plural, prev, d.path)
		}
		if r.Name == r.Plural && !r.Skip {
			return nil, nil, fmt.Errorf("%s: singular and plural names are both %q: add a plural override", d.path, r.Name)
		}
		seenNames[r.Name] = d.path
		seenPlural[r.Plural] = d.path
		readSchema := d.detail.Get.ResponseSchema()
		if readSchema == nil {
			return nil, nil, fmt.Errorf("%s: no read schema", d.path)
		}
		_, r.ReadSchema = doc.Resolve(readSchema)
		b.readSchemaToName[r.ReadSchema] = r.Name
		resources = append(resources, r)
	}

	// Pass 2: attributes.
	for _, r := range resources {
		if err := b.buildResource(r); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", r.Path, err)
		}
	}
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].App != resources[j].App {
			return resources[i].App < resources[j].App
		}
		return resources[i].Name < resources[j].Name
	})
	return resources, b.Warnings, nil
}

// lookupOverride returns the override for "app/segment", falling back to
// "segment" and then to an empty override.
func lookupOverride(ov map[string]*overrides.Resource, app, seg string) *overrides.Resource {
	if o := ov[app+"/"+seg]; o != nil {
		return o
	}
	if o := ov[seg]; o != nil {
		return o
	}
	return &overrides.Resource{}
}

func (b *Builder) warn(format string, args ...any) {
	b.Warnings = append(b.Warnings, fmt.Sprintf(format, args...))
}

func (b *Builder) buildResource(r *model.Resource) error {
	doc := b.Doc
	o := lookupOverride(b.Overrides, r.App, r.PathSegment)
	item := doc.Paths[r.Path]
	detail := doc.Paths[r.Path+"{id}/"]
	list := item.Get

	// Client identifiers from operationIds: "dcim_sites_list" -> DcimSites.
	opID := list.OperationID
	if !strings.HasSuffix(opID, "_list") {
		return fmt.Errorf("unexpected list operationId %q", opID)
	}
	r.OpPrefix = naming.GoMethod(strings.TrimSuffix(opID, "_list"))
	if len(list.Tags) == 0 {
		return fmt.Errorf("list operation has no tag")
	}
	r.Service = naming.GoType(list.Tags[0]) + "API"
	r.Description = strings.TrimSpace(strings.TrimPrefix(list.Description, "Get a list of "))
	if o.Description != "" {
		r.Description = o.Description
	}
	r.ReadType = naming.GoType(r.ReadSchema)

	readSchema := doc.Components.Schemas[r.ReadSchema]
	var createSchema, patchSchema *openapi.Schema
	if !r.DataSourceOnly {
		cs := item.Post.BodySchema()
		if cs == nil {
			return fmt.Errorf("no create body")
		}
		createSchema, r.CreateSchema = doc.Resolve(cs)
		ps := detail.Patch.BodySchema()
		if ps == nil {
			return fmt.Errorf("no patch body")
		}
		patchSchema, r.PatchSchema = doc.Resolve(ps)
		r.CreateType = naming.GoType(r.CreateSchema)
		r.PatchType = naming.GoType(r.PatchSchema)
		for _, p := range createSchema.OrderedProperties() {
			if createSchema.IsRequired(p) {
				r.CreateRequired = append(r.CreateRequired, p)
			}
		}
	}
	_ = patchSchema

	// id first.
	r.Attrs = append(r.Attrs, model.Attr{Name: "id", JSON: "id", GoField: "Id", Kind: model.KindInt, ReadOnly: true, Computed: true,
		ReadKind: model.ReadScalar, ReadGoField: "Id", Description: "The numeric ID of the object in NetBox."})

	if r.DataSourceOnly {
		for _, pname := range readSchema.OrderedProperties() {
			if pname == "id" {
				continue
			}
			if ao := o.Attributes[pname]; ao != nil && ao.Skip {
				continue
			}
			ro, ok := b.classifyReadOnly(pname, readSchema.Properties[pname])
			if !ok {
				b.warn("%s.%s: read-only property skipped", r.Name, pname)
				continue
			}
			r.Attrs = append(r.Attrs, ro)
		}
	} else {
		for _, pname := range createSchema.OrderedProperties() {
			if pname == "id" {
				continue
			}
			prop := createSchema.Properties[pname]
			var ao *overrides.Attribute
			if o.Attributes != nil {
				ao = o.Attributes[pname]
			}
			if ao != nil && ao.Skip {
				continue
			}
			var readProp *openapi.Schema
			if readSchema != nil {
				readProp = readSchema.Properties[pname]
			}
			attr, err := b.classify(r, pname, prop, createSchema.IsRequired(pname), readProp, ao)
			if err != nil {
				return err
			}
			if attr == nil {
				continue
			}
			r.Attrs = append(r.Attrs, *attr)
		}

		// Read-only extras: url, display, created, last_updated on the resource;
		// everything else that is readable-only goes to ReadOnlyAttrs (data sources).
		for _, pname := range readSchema.OrderedProperties() {
			if pname == "id" || r.Attr(pname) != nil || r.Attr(pname+"_id") != nil || r.Attr(naming.Singular(pname)+"_ids") != nil {
				continue
			}
			if createSchema.Properties[pname] != nil {
				continue // a skipped write attribute
			}
			prop := readSchema.Properties[pname]
			ro, ok := b.classifyReadOnly(pname, prop)
			if !ok {
				continue
			}
			switch pname {
			case "url", "display", "created", "last_updated":
				r.Attrs = append(r.Attrs, ro)
			default:
				r.ReadOnlyAttrs = append(r.ReadOnlyAttrs, ro)
			}
		}
	}

	// Filters.
	for _, p := range list.Parameters {
		if p.In != "query" || nonFilterParams[p.Name] {
			continue
		}
		r.Filters = append(r.Filters, model.Filter{Name: p.Name, Description: strings.TrimSpace(p.Description)})
	}
	sort.Slice(r.Filters, func(i, j int) bool { return r.Filters[i].Name < r.Filters[j].Name })

	// Lookups.
	filterNames := map[string]bool{}
	for _, f := range r.Filters {
		filterNames[f.Name] = true
	}
	if o.Lookups != nil {
		r.Lookups = o.Lookups
	} else {
		for _, cand := range []string{"name", "slug"} {
			if a := r.Attr(cand); a != nil && filterNames[cand] {
				r.Lookups = append(r.Lookups, cand)
			}
		}
	}
	for _, l := range r.Lookups {
		if r.Attr(l) == nil {
			return fmt.Errorf("lookup %q is not an attribute", l)
		}
	}

	// Flags, deps.
	depSet := map[string]bool{}
	for _, a := range r.Attrs {
		switch a.Kind {
		case model.KindTags:
			r.HasTags = true
		case model.KindCustomFields:
			r.HasCustomFields = true
		}
		collectTargets(a, depSet)
	}
	delete(depSet, r.Name)
	for d := range depSet {
		r.Deps = append(r.Deps, d)
	}
	sort.Strings(r.Deps)

	// Tests.
	r.Test.Serial = o.SerialTest
	r.Test.ImportVerifyIgnore = o.ImportVerifyIgnore
	if o.Test != nil {
		r.Test.Basic = o.Test.Basic
		r.Test.Update = o.Test.Update
		r.Test.Skip = o.Test.Skip
		r.Test.Checks = o.Test.Checks
		r.Test.ImportVerifyIgnore = append(r.Test.ImportVerifyIgnore, o.Test.ImportVerifyIgnore...)
	}
	return nil
}

func collectTargets(a model.Attr, set map[string]bool) {
	if a.Target != "" {
		set[a.Target] = true
	}
	for _, n := range a.Nested {
		collectTargets(n, set)
	}
}

// targetFromRead derives the FK target Terraform name from a read component
// ("BriefTenant" -> tenant, "NestedInterface" -> interface).
func (b *Builder) targetFromRead(component string) string {
	for _, prefix := range []string{"Brief", "Nested", ""} {
		if strings.HasPrefix(component, prefix) {
			base := strings.TrimPrefix(component, prefix)
			if n, ok := b.readSchemaToName[base]; ok {
				return n
			}
		}
	}
	return ""
}

// classify maps one request property to an attribute.
func (b *Builder) classify(r *model.Resource, pname string, prop *openapi.Schema, required bool, readProp *openapi.Schema, ao *overrides.Attribute) (*model.Attr, error) {
	doc := b.Doc
	a := &model.Attr{Name: pname, JSON: pname, GoField: naming.GoField(pname), Required: required, Nullable: prop.Nullable,
		Description: strings.TrimSpace(prop.Description), WriteOnly: prop.WriteOnly}
	a.MaxLength, a.MinLength, a.Pattern = prop.MaxLength, prop.MinLength, prop.Pattern
	rs, refName := doc.Resolve(prop)
	if rs == nil {
		return nil, fmt.Errorf("%s: unresolvable schema", pname)
	}
	if prop.Ref != "" || len(prop.AllOf) > 0 {
		// nested single object (none exist in 4.7)
		if rs.Type == "object" && len(rs.Properties) > 0 && !isChoiceObject(rs) {
			return nil, fmt.Errorf("%s: nested object %s is not supported; skip it in overrides", pname, refName)
		}
	}

	// Read side.
	b.pairRead(a, readProp)

	kind := model.Kind("")
	if ao != nil && ao.Kind != "" {
		kind = model.Kind(ao.Kind)
	}
	switch {
	case kind != "":
	case pname == "tags" && rs.Type == "array" && rs.Items != nil && openapi.RefName(rs.Items.Ref) == "NestedTagRequest":
		kind = model.KindTags
	case pname == "custom_fields":
		kind = model.KindCustomFields
	case len(rs.OneOf) > 0:
		ints, refs, strs := 0, 0, 0
		for _, alt := range rs.OneOf {
			switch {
			case alt.Type == "integer":
				ints++
			case alt.Ref != "" || len(alt.AllOf) > 0:
				refs++
			case alt.Type == "string":
				strs++
			}
		}
		switch {
		case ints == 1 && refs == 1:
			kind = model.KindFK
		case strs == len(rs.OneOf):
			// oneOf [string(pattern), string(maxLength 0)]: blank or pattern.
			kind = model.KindString
			for _, alt := range rs.OneOf {
				if alt.MaxLength != nil && *alt.MaxLength > 0 {
					a.MaxLength = alt.MaxLength
					if alt.Pattern != "" {
						a.Pattern = "^$|" + alt.Pattern
					}
				}
			}
		default:
			return nil, fmt.Errorf("%s: unsupported oneOf", pname)
		}
	case rs.Type == "integer":
		a.Int64 = rs.Format == "int64"
		switch {
		case a.ReadKind == model.ReadBrief:
			kind = model.KindFK
		case len(rs.Enum) > 0:
			kind = model.KindChoiceInt
			a.EnumInts = rs.EnumInts()
		default:
			kind = model.KindInt
		}
	case rs.Type == "string":
		switch {
		case rs.Format == "binary":
			b.warn("%s.%s: binary field skipped", r.Name, pname)
			return nil, nil
		case rs.Format == "date-time":
			kind = model.KindDateTime
		case len(rs.Enum) > 0:
			kind = model.KindChoice
			a.Enum = rs.EnumStrings()
		default:
			kind = model.KindString
			a.DateOnly = rs.Format == "date"
		}
	case rs.Type == "number":
		kind = model.KindFloat
		a.Float32 = rs.Format != "double"
	case rs.Type == "boolean":
		kind = model.KindBool
	case rs.Type == "array":
		items := rs.Items
		if items == nil {
			return nil, fmt.Errorf("%s: array without items", pname)
		}
		irs, iref := doc.Resolve(items)
		switch {
		case irs == nil:
			return nil, fmt.Errorf("%s: unresolvable items", pname)
		case irs.Type == "integer":
			a.Int64 = irs.Format == "int64"
			if a.ReadKind == model.ReadBriefList {
				kind = model.KindFKList
			} else {
				kind = model.KindIntList
			}
		case irs.Type == "string":
			kind = model.KindStringList
			if len(irs.Enum) > 0 {
				a.Enum = irs.EnumStrings()
			}
		case irs.Type == "array":
			inner, _ := doc.Resolve(irs.Items)
			if inner != nil && inner.Type == "integer" {
				kind = model.KindIntRangeList
				a.Int64 = inner.Format == "int64"
			} else {
				kind = model.KindAnyList
			}
		case irs.Type == "object" || len(irs.Properties) > 0:
			kind = model.KindNestedList
			a.NestedReqType = naming.GoType(iref)
			if err := b.buildNested(r, a, irs, readProp, ao); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%s: unsupported array items type %q", pname, irs.Type)
		}
	case rs.Type == "object" || rs.Type == "":
		if len(rs.Properties) > 0 && !rs.HasAdditionalProperties() {
			return nil, fmt.Errorf("%s: inline object not supported", pname)
		}
		kind = model.KindJSON
		if ap := additionalPropertiesType(rs); ap == "string" {
			a.StringMap = true
		}
	default:
		return nil, fmt.Errorf("%s: unsupported type %q", pname, rs.Type)
	}
	a.Kind = kind

	// Terraform naming.
	switch kind {
	case model.KindFK:
		if !strings.HasSuffix(a.Name, "_id") {
			a.Name += "_id"
		}
	case model.KindFKList:
		a.Name = naming.Singular(a.Name) + "_ids"
	}
	if kind == model.KindFK && a.Target == "" && ao == nil {
		b.warn("%s.%s: FK target unknown (bare integer); set attributes.%s.target in overrides", r.Name, pname, pname)
	}

	// Terraform optionality.
	if !a.Required {
		switch kind {
		case model.KindBool, model.KindChoice, model.KindChoiceInt:
			a.Computed = true
		case model.KindString:
			if !a.Nullable {
				a.Computed = true
				// Default to "" (and always send it) only when NetBox accepts a
				// blank value; otherwise behave like a server-defaulted field.
				a.DefaultEmptyString = blankAllowed(a)
			}
		case model.KindInt, model.KindFloat:
			if !a.Nullable {
				a.Computed = true
			}
		case model.KindFKList, model.KindIntList, model.KindStringList, model.KindTags:
			a.Computed = true
			a.DefaultEmptySet = true
		}
	}
	if a.WriteOnly {
		a.Computed = false
	}
	if kind == model.KindDateTime && !a.Required {
		a.Computed = true
	}

	// Overrides.
	if ao != nil {
		if ao.Name != "" {
			a.Name = ao.Name
		}
		if ao.Target != "" {
			a.Target = ao.Target
		}
		a.RequiresReplace = ao.RequiresReplace
		a.Sensitive = ao.Sensitive
		if ao.Computed != nil {
			a.Computed = *ao.Computed
			if !a.Computed {
				a.DefaultEmptyString, a.DefaultEmptySet = false, false
			}
		}
		if ao.Optional != nil && *ao.Optional {
			a.Required = false
		}
		if ao.Precision > 0 {
			a.Precision = ao.Precision
		}
		a.OrderedList = ao.OrderedList
		if ao.Description != "" {
			a.Description = ao.Description
		}
		if len(ao.Enum) > 0 {
			a.Enum = ao.Enum
		}
	}
	if a.Description == "" {
		a.Description = defaultDescription(a)
	}
	return a, nil
}

// additionalPropertiesType returns the declared type of additionalProperties ("" when free-form).
func additionalPropertiesType(s *openapi.Schema) string {
	if len(s.AdditionalProperties) == 0 {
		return ""
	}
	var sub struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(s.AdditionalProperties, &sub); err != nil {
		return ""
	}
	return sub.Type
}

// blankAllowed reports whether "" is a valid value for a string property.
func blankAllowed(a *model.Attr) bool {
	if a.MinLength != nil && *a.MinLength > 0 {
		return false
	}
	if a.Pattern == "" {
		return true
	}
	re, err := regexp.Compile(a.Pattern)
	if err != nil {
		return true
	}
	return re.MatchString("")
}

func defaultDescription(a *model.Attr) string {
	switch a.Kind {
	case model.KindFK:
		if a.Target != "" {
			return fmt.Sprintf("ID of the %s (`netbox_%s`).", naming.Title(a.Target), a.Target)
		}
		return fmt.Sprintf("ID of the related %s.", naming.Title(strings.TrimSuffix(a.Name, "_id")))
	case model.KindFKList:
		if a.Target != "" {
			return fmt.Sprintf("IDs of the assigned %s (`netbox_%s`).", naming.Title(naming.Singular(strings.TrimSuffix(a.Name, "_ids"))), a.Target)
		}
		return fmt.Sprintf("IDs of the assigned %s.", naming.Title(strings.TrimSuffix(a.Name, "_ids")))
	case model.KindTags:
		return "Slugs of the tags assigned to this object."
	case model.KindCustomFields:
		return "Custom field values as a JSON object (`jsonencode({...})`). Only keys present in the configuration are tracked."
	case model.KindJSON:
		return fmt.Sprintf("%s as a JSON document (`jsonencode({...})`).", naming.Title(a.JSON))
	}
	return naming.Title(a.JSON) + "."
}

func isChoiceObject(s *openapi.Schema) bool {
	if s == nil || s.Type != "object" || len(s.Properties) != 2 {
		return false
	}
	_, v := s.Properties["value"]
	_, l := s.Properties["label"]
	return v && l
}

// pairRead sets the read-side classification of an attribute.
func (b *Builder) pairRead(a *model.Attr, readProp *openapi.Schema) {
	doc := b.Doc
	a.ReadGoField = a.GoField
	if readProp == nil {
		a.ReadKind = model.ReadAbsent
		return
	}
	rs, refName := doc.Resolve(readProp)
	if rs == nil {
		a.ReadKind = model.ReadAbsent
		return
	}
	switch {
	case len(rs.OneOf) > 0:
		// oneOf of strings (colour, dns_name, email): the client reads a plain string.
		a.ReadKind = model.ReadScalar
	case isChoiceObject(rs):
		a.ReadKind = model.ReadChoice
		if v := rs.Properties["value"]; v != nil && v.Type == "integer" {
			a.ReadChoiceInt = true
		}
	case rs.Type == "object" && len(rs.Properties) > 0:
		if _, hasID := rs.Properties["id"]; hasID {
			a.ReadKind = model.ReadBrief
			a.Target = b.targetFromRead(refName)
		} else {
			a.ReadKind = model.ReadJSON
		}
	case rs.Type == "object" || rs.Type == "":
		if rs.HasAdditionalProperties() {
			a.ReadKind = model.ReadMap
		} else {
			a.ReadKind = model.ReadJSON
		}
	case rs.Type == "array":
		irs, iref := doc.Resolve(rs.Items)
		switch {
		case irs == nil:
			a.ReadKind = model.ReadJSON
		case iref == "NestedTag":
			a.ReadKind = model.ReadTags
		case irs.Type == "integer":
			a.ReadKind = model.ReadIntList
		case irs.Type == "string":
			a.ReadKind = model.ReadStringList
		case irs.Type == "array":
			inner, _ := doc.Resolve(irs.Items)
			if inner != nil && inner.Type == "integer" {
				a.ReadKind = model.ReadIntRange
			} else {
				a.ReadKind = model.ReadAnyList
			}
		case irs.Type == "object" || len(irs.Properties) > 0:
			if _, hasID := irs.Properties["id"]; hasID {
				a.ReadKind = model.ReadBriefList
				a.Target = b.targetFromRead(iref)
			} else {
				a.ReadKind = model.ReadNestedList
				a.NestedRspType = naming.GoType(iref)
			}
		default:
			a.ReadKind = model.ReadJSON
		}
	default:
		a.ReadKind = model.ReadScalar
	}
}

// buildNested classifies the item properties of a nested object list.
func (b *Builder) buildNested(r *model.Resource, a *model.Attr, itemReq *openapi.Schema, readProp *openapi.Schema, ao *overrides.Attribute) error {
	doc := b.Doc
	var itemRead *openapi.Schema
	if readProp != nil {
		if rrs, _ := doc.Resolve(readProp); rrs != nil && rrs.Type == "array" && rrs.Items != nil {
			itemRead, _ = doc.Resolve(rrs.Items)
		}
	}
	a.Target = ""
	if a.ReadKind != model.ReadNestedList {
		// Read side is a brief list or absent: still treat as nested list, reading what we can.
		if itemRead != nil {
			a.ReadKind = model.ReadNestedList
			if readProp != nil {
				if rrs, _ := doc.Resolve(readProp); rrs != nil && rrs.Items != nil {
					_, iref := doc.Resolve(rrs.Items)
					a.NestedRspType = naming.GoType(iref)
				}
			}
		}
	}
	for _, pn := range itemReq.OrderedProperties() {
		if itemReq.IsRequired(pn) {
			a.NestedItemRequired = append(a.NestedItemRequired, pn)
		}
	}
	for _, pn := range itemReq.OrderedProperties() {
		p := itemReq.Properties[pn]
		var nao *overrides.Attribute
		if ao != nil && ao.Nested != nil {
			nao = ao.Nested[pn]
		}
		if nao != nil && nao.Skip {
			continue
		}
		var rp *openapi.Schema
		if itemRead != nil {
			rp = itemRead.Properties[pn]
		}
		na, err := b.classify(r, pn, p, itemReq.IsRequired(pn), rp, nao)
		if err != nil {
			return fmt.Errorf("%s.%s: %w", a.JSON, pn, err)
		}
		if na == nil {
			continue
		}
		if na.Kind == model.KindNestedList {
			return fmt.Errorf("%s.%s: nested lists inside nested lists are not supported", a.JSON, pn)
		}
		// Inside nested objects there are no server defaults to track.
		na.Computed = false
		na.DefaultEmptyString = false
		na.DefaultEmptySet = false
		a.Nested = append(a.Nested, *na)
	}
	return nil
}

// classifyReadOnly maps a read-only property (counts, display_url, ...) to a computed attribute.
func (b *Builder) classifyReadOnly(pname string, prop *openapi.Schema) (model.Attr, bool) {
	doc := b.Doc
	a := model.Attr{Name: strings.TrimLeft(pname, "_"), JSON: pname, GoField: naming.GoField(pname), ReadGoField: naming.GoField(pname), ReadOnly: true, Computed: true,
		Description: strings.TrimSpace(prop.Description), Nullable: prop.Nullable}
	rs, refName := doc.Resolve(prop)
	if rs == nil {
		return a, false
	}
	switch {
	case isChoiceObject(rs):
		a.ReadKind = model.ReadChoice
		if v := rs.Properties["value"]; v != nil && v.Type == "integer" {
			a.ReadChoiceInt = true
			a.Kind = model.KindInt
		} else {
			a.Kind = model.KindString
		}
	case rs.Type == "object" && len(rs.Properties) > 0:
		if _, hasID := rs.Properties["id"]; hasID {
			a.ReadKind = model.ReadBrief
			a.Kind = model.KindFK
			a.Name += "_id"
			a.Target = b.targetFromRead(refName)
		} else {
			return a, false
		}
	case rs.Type == "integer":
		a.Kind, a.ReadKind, a.Int64 = model.KindInt, model.ReadScalar, rs.Format == "int64"
	case rs.Type == "number":
		a.Kind, a.ReadKind, a.Float32 = model.KindFloat, model.ReadScalar, rs.Format != "double"
	case rs.Type == "boolean":
		a.Kind, a.ReadKind = model.KindBool, model.ReadScalar
	case rs.Type == "string":
		a.ReadKind = model.ReadScalar
		switch rs.Format {
		case "date-time":
			a.Kind = model.KindDateTime
		case "binary":
			return a, false
		default:
			a.Kind = model.KindString
		}
	case rs.Type == "array":
		irs, iref := doc.Resolve(rs.Items)
		switch {
		case irs == nil:
			return a, false
		case iref == "NestedTag":
			a.Kind, a.ReadKind = model.KindTags, model.ReadTags
		case irs.Type == "integer":
			a.Kind, a.ReadKind, a.Int64 = model.KindIntList, model.ReadIntList, irs.Format == "int64"
		case irs.Type == "string":
			a.Kind, a.ReadKind = model.KindStringList, model.ReadStringList
		case irs.Type == "object" || len(irs.Properties) > 0:
			if _, hasID := irs.Properties["id"]; hasID {
				a.Kind, a.ReadKind = model.KindFKList, model.ReadBriefList
				a.Name = naming.Singular(a.Name) + "_ids"
				a.Target = b.targetFromRead(iref)
			} else {
				return a, false
			}
		default:
			return a, false
		}
	default:
		// untyped (config_context, generic objects): expose as JSON
		if rs.Type == "" || rs.Type == "object" {
			a.Kind, a.ReadKind = model.KindJSON, model.ReadJSON
		} else {
			return a, false
		}
	}
	if a.Description == "" {
		a.Description = naming.Title(pname) + "."
	}
	return a, true
}
