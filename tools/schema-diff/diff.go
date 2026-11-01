package main

import (
	"fmt"
	"slices"
	"strings"
)

// Severity of a change for users of the provider.
type Severity int

// Severities, most severe first.
const (
	// Breaking changes can make an existing configuration fail to validate,
	// plan changes it did not plan before, or break references to the value.
	Breaking Severity = iota
	// Additive changes only add things an existing configuration cannot use yet.
	Additive
	// Notice changes are visible to users but leave configurations working
	// (sensitivity, deprecations, schema version bumps).
	Notice
)

func (s Severity) String() string {
	switch s {
	case Breaking:
		return "breaking"
	case Additive:
		return "additive"
	default:
		return "notice"
	}
}

// Kind of top-level schema object.
type Kind int

// Kinds in report order.
const (
	KindProvider Kind = iota
	KindResource
	KindDataSource
	KindEphemeral
	KindListResource
	KindAction
	KindFunction
	KindIdentity
)

func (k Kind) String() string {
	return [...]string{"provider", "resource", "data source", "ephemeral resource", "list resource", "action", "function", "resource identity"}[k]
}

// Change is one difference between the two schemas.
type Change struct {
	Severity Severity
	Kind     Kind
	Name     string // type name, e.g. netbox_site ("" for the provider block)
	Path     string // attribute or block path, e.g. rear_ports.position ("" for the object itself)
	Detail   string
}

type differ struct {
	changes []Change
	// descriptions counts attributes whose description changed. They are not
	// classified, but the count hints at changes the schema cannot express,
	// such as allowed values (validators are not part of the schema).
	descriptions int
}

func (d *differ) add(sev Severity, kind Kind, name, path, format string, args ...any) {
	d.changes = append(d.changes, Change{sev, kind, name, path, fmt.Sprintf(format, args...)})
}

// Result of a comparison.
type Result struct {
	Changes []Change
	// DescriptionChanges is the number of attributes present in both schemas
	// whose description differs.
	DescriptionChanges int
}

// Diff compares two provider schemas and returns the changes in report order.
func Diff(before, after *providerSchema) Result {
	d := &differ{}
	if before.Provider != nil || after.Provider != nil {
		d.schema(KindProvider, "", orEmpty(before.Provider), orEmpty(after.Provider))
	}
	d.schemas(KindResource, before.ResourceSchemas, after.ResourceSchemas)
	d.schemas(KindDataSource, before.DataSourceSchemas, after.DataSourceSchemas)
	d.schemas(KindEphemeral, before.EphemeralResourceSchemas, after.EphemeralResourceSchemas)
	d.schemas(KindListResource, before.ListResourceSchemas, after.ListResourceSchemas)
	d.schemas(KindAction, before.ActionSchemas, after.ActionSchemas)
	d.functions(before.Functions, after.Functions)
	d.identities(before, after)
	slices.SortStableFunc(d.changes, func(a, b Change) int {
		if a.Severity != b.Severity {
			return int(a.Severity) - int(b.Severity)
		}
		if a.Kind != b.Kind {
			return int(a.Kind) - int(b.Kind)
		}
		if c := strings.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		return strings.Compare(a.Path, b.Path)
	})
	return Result{Changes: d.changes, DescriptionChanges: d.descriptions}
}

func orEmpty(s *schema) *schema {
	if s == nil {
		return &schema{Block: &block{}}
	}
	return s
}

func (d *differ) schemas(kind Kind, before, after map[string]*schema) {
	for _, name := range unionKeys(before, after) {
		o, n := before[name], after[name]
		switch {
		case n == nil:
			d.add(Breaking, kind, name, "", "removed")
		case o == nil:
			d.add(Additive, kind, name, "", "new %s", kind)
		default:
			d.schema(kind, name, o, n)
		}
	}
}

func (d *differ) schema(kind Kind, name string, before, after *schema) {
	switch {
	case after.Version < before.Version:
		// Terraform refuses state written by a newer schema version.
		d.add(Breaking, kind, name, "", "schema version decreased from %d to %d", before.Version, after.Version)
	case after.Version > before.Version:
		d.add(Notice, kind, name, "", "schema version increased from %d to %d (state is upgraded on the next refresh)", before.Version, after.Version)
	}
	ob, nb := before.Block, after.Block
	if ob == nil {
		ob = &block{}
	}
	if nb == nil {
		nb = &block{}
	}
	if !ob.Deprecated && nb.Deprecated {
		d.add(Notice, kind, name, "", "deprecated")
	} else if ob.Deprecated && !nb.Deprecated {
		d.add(Notice, kind, name, "", "no longer deprecated")
	}
	d.block(kind, name, "", ob, nb)
}

func join(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func (d *differ) block(kind Kind, name, prefix string, before, after *block) {
	d.attributes(kind, name, prefix, before.Attributes, after.Attributes, false)
	for _, bn := range unionKeys(before.BlockTypes, after.BlockTypes) {
		path := join(prefix, bn)
		o, n := before.BlockTypes[bn], after.BlockTypes[bn]
		switch {
		case n == nil:
			d.add(Breaking, kind, name, path, "nested block removed")
		case o == nil:
			if n.MinItems > 0 {
				d.add(Breaking, kind, name, path, "new required nested block (min %d)", n.MinItems)
			} else {
				d.add(Additive, kind, name, path, "new optional nested block")
			}
		default:
			if o.NestingMode != n.NestingMode {
				d.add(Breaking, kind, name, path, "nested block mode changed from %s to %s", o.NestingMode, n.NestingMode)
				continue
			}
			d.itemLimits(kind, name, path, "nested block", o.MinItems, o.MaxItems, n.MinItems, n.MaxItems)
			ob, nb := o.Block, n.Block
			if ob == nil {
				ob = &block{}
			}
			if nb == nil {
				nb = &block{}
			}
			if !ob.Deprecated && nb.Deprecated {
				d.add(Notice, kind, name, path, "nested block deprecated")
			}
			d.block(kind, name, path, ob, nb)
		}
	}
}

func (d *differ) itemLimits(kind Kind, name, path, what string, oMin, oMax, nMin, nMax uint64) {
	switch {
	case nMin > oMin:
		d.add(Breaking, kind, name, path, "%s needs at least %d items (was %d)", what, nMin, oMin)
	case nMin < oMin:
		d.add(Additive, kind, name, path, "%s needs at least %d items (was %d)", what, nMin, oMin)
	}
	// 0 means unlimited.
	switch {
	case nMax != 0 && (oMax == 0 || nMax < oMax):
		d.add(Breaking, kind, name, path, "%s allows at most %d items (was %s)", what, nMax, limit(oMax))
	case oMax != 0 && (nMax == 0 || nMax > oMax):
		d.add(Additive, kind, name, path, "%s allows at most %s items (was %d)", what, limit(nMax), oMax)
	}
}

func limit(n uint64) string {
	if n == 0 {
		return "unlimited"
	}
	return fmt.Sprint(n)
}

// attributes compares two attribute sets. readOnly is true below a
// computed-only nested attribute, where users cannot set anything, so only
// removals and type changes matter.
func (d *differ) attributes(kind Kind, name, prefix string, before, after map[string]*attribute, readOnly bool) {
	for _, an := range unionKeys(before, after) {
		path := join(prefix, an)
		o, n := before[an], after[an]
		switch {
		case n == nil:
			d.add(Breaking, kind, name, path, "attribute removed")
		case o == nil:
			switch {
			case readOnly || mode(n) == modeComputed:
				d.add(Additive, kind, name, path, "new computed attribute")
			case mode(n) == modeRequired:
				d.add(Breaking, kind, name, path, "new required attribute")
			default:
				d.add(Additive, kind, name, path, "new optional attribute")
			}
		default:
			d.attribute(kind, name, path, o, n, readOnly)
		}
	}
}

type attrMode int

const (
	modeRequired attrMode = iota
	modeOptional
	modeOptionalComputed
	modeComputed
)

func (m attrMode) String() string {
	return [...]string{"required", "optional", "optional+computed", "computed-only"}[m]
}

func mode(a *attribute) attrMode {
	switch {
	case a.Required:
		return modeRequired
	case a.Optional && a.Computed:
		return modeOptionalComputed
	case a.Optional:
		return modeOptional
	default:
		return modeComputed
	}
}

// modeChange classifies a change of an attribute's required/optional/computed
// flags. See the package documentation for the reasoning behind each case.
func modeChange(o, n attrMode) (Severity, string) {
	what := fmt.Sprintf("%s -> %s", o, n)
	switch {
	case n == modeRequired:
		return Breaking, what + ": configurations that omit it fail"
	case n == modeComputed:
		return Breaking, what + ": configurations that set it fail"
	case o == modeRequired, o == modeComputed && n == modeOptionalComputed:
		return Additive, what
	case o == modeOptionalComputed && n == modeOptional:
		return Breaking, what + ": an omitted value is now null instead of the server value, so existing resources plan a change"
	case o == modeComputed && n == modeOptional:
		return Breaking, what + ": the value is no longer filled in unless configured"
	default: // optional -> optional+computed
		return Notice, what + ": an omitted value is now filled in by the provider instead of null"
	}
}

func (d *differ) attribute(kind Kind, name, path string, o, n *attribute, readOnly bool) {
	if o.Description != n.Description {
		d.descriptions++
	}
	bothNested := o.NestedType != nil && n.NestedType != nil
	switch {
	case bothNested && o.NestedType.NestingMode != n.NestedType.NestingMode:
		d.add(Breaking, kind, name, path, "type changed from %s to %s", describeType(o), describeType(n))
		return
	case bothNested:
		// Same nesting mode: child differences are reported on the children.
	case impliedType(o) != impliedType(n):
		d.add(Breaking, kind, name, path, "type changed from %s to %s", describeType(o), describeType(n))
		return
	default:
		if (o.NestedType == nil) != (n.NestedType == nil) {
			d.add(Notice, kind, name, path, "changed from %s to %s with the same value type", describeType(o), describeType(n))
		}
	}

	if om, nm := mode(o), mode(n); om != nm && !readOnly {
		sev, detail := modeChange(om, nm)
		d.add(sev, kind, name, path, "%s", detail)
	}
	switch {
	case !o.WriteOnly && n.WriteOnly:
		d.add(Breaking, kind, name, path, "now write-only: the value is no longer stored in state or readable")
	case o.WriteOnly && !n.WriteOnly:
		d.add(Notice, kind, name, path, "no longer write-only: the value is stored in state")
	}
	switch {
	case !o.Sensitive && n.Sensitive:
		d.add(Notice, kind, name, path, "now sensitive (root module outputs that expose it need `sensitive = true`)")
	case o.Sensitive && !n.Sensitive:
		d.add(Notice, kind, name, path, "no longer sensitive")
	}
	switch {
	case !o.Deprecated && n.Deprecated:
		d.add(Notice, kind, name, path, "deprecated")
	case o.Deprecated && !n.Deprecated:
		d.add(Notice, kind, name, path, "no longer deprecated")
	}

	if bothNested {
		d.itemLimits(kind, name, path, "nested attribute", o.NestedType.MinItems, o.NestedType.MaxItems, n.NestedType.MinItems, n.NestedType.MaxItems)
		childReadOnly := readOnly || mode(n) == modeComputed
		d.attributes(kind, name, path, o.NestedType.Attributes, n.NestedType.Attributes, childReadOnly)
	}
}

func (d *differ) functions(before, after map[string]*function) {
	for _, name := range unionKeys(before, after) {
		o, n := before[name], after[name]
		switch {
		case n == nil:
			d.add(Breaking, KindFunction, name, "", "removed")
			continue
		case o == nil:
			d.add(Additive, KindFunction, name, "", "new function")
			continue
		}
		if canonicalType(o.ReturnType) != canonicalType(n.ReturnType) {
			d.add(Breaking, KindFunction, name, "", "return type changed from %s to %s", typeString(o.ReturnType), typeString(n.ReturnType))
		}
		if len(o.Parameters) != len(n.Parameters) {
			d.add(Breaking, KindFunction, name, "", "takes %d parameters (was %d)", len(n.Parameters), len(o.Parameters))
		} else {
			for i := range o.Parameters {
				d.parameter(name, fmt.Sprintf("parameter %d (%s)", i+1, n.Parameters[i].Name), o.Parameters[i], n.Parameters[i])
			}
		}
		switch {
		case o.VariadicParameter != nil && n.VariadicParameter == nil:
			d.add(Breaking, KindFunction, name, "", "variadic parameter removed")
		case o.VariadicParameter == nil && n.VariadicParameter != nil:
			d.add(Additive, KindFunction, name, "", "new variadic parameter")
		case o.VariadicParameter != nil:
			d.parameter(name, "variadic parameter", o.VariadicParameter, n.VariadicParameter)
		}
		if o.DeprecationMessage == "" && n.DeprecationMessage != "" {
			d.add(Notice, KindFunction, name, "", "deprecated")
		}
	}
}

func (d *differ) parameter(fn, label string, o, n *parameter) {
	if canonicalType(o.Type) != canonicalType(n.Type) {
		d.add(Breaking, KindFunction, fn, "", "%s type changed from %s to %s", label, typeString(o.Type), typeString(n.Type))
	}
	if o.IsNullable && !n.IsNullable {
		d.add(Breaking, KindFunction, fn, "", "%s no longer accepts null", label)
	}
}

// identities compares resource identity schemas of resources present in both
// documents (a removed or new resource is already reported as such).
func (d *differ) identities(before, after *providerSchema) {
	for _, name := range unionKeys(before.ResourceIdentitySchemas, after.ResourceIdentitySchemas) {
		if before.ResourceSchemas[name] == nil || after.ResourceSchemas[name] == nil {
			continue
		}
		o, n := before.ResourceIdentitySchemas[name], after.ResourceIdentitySchemas[name]
		switch {
		case n == nil:
			d.add(Breaking, KindIdentity, name, "", "removed: import blocks using `identity` fail")
			continue
		case o == nil:
			d.add(Additive, KindIdentity, name, "", "new resource identity")
			continue
		}
		switch {
		case n.Version < o.Version:
			d.add(Breaking, KindIdentity, name, "", "identity version decreased from %d to %d", o.Version, n.Version)
		case n.Version > o.Version:
			d.add(Notice, KindIdentity, name, "", "identity version increased from %d to %d", o.Version, n.Version)
		}
		for _, an := range unionKeys(o.Attributes, n.Attributes) {
			oa, na := o.Attributes[an], n.Attributes[an]
			switch {
			case na == nil:
				d.add(Breaking, KindIdentity, name, an, "identity attribute removed")
			case oa == nil && na.RequiredForImport:
				d.add(Breaking, KindIdentity, name, an, "new identity attribute required for import")
			case oa == nil:
				d.add(Additive, KindIdentity, name, an, "new identity attribute")
			case canonicalType(oa.Type) != canonicalType(na.Type):
				d.add(Breaking, KindIdentity, name, an, "type changed from %s to %s", typeString(oa.Type), typeString(na.Type))
			case !oa.RequiredForImport && na.RequiredForImport:
				d.add(Breaking, KindIdentity, name, an, "now required for import")
			case oa.RequiredForImport && !na.RequiredForImport:
				d.add(Additive, KindIdentity, name, an, "no longer required for import")
			}
		}
	}
}
