// Package model defines the intermediate representation shared by the
// generator's builder and renderers: one Resource per NetBox CRUD collection.
package model

import "strings"

// Kind classifies how an API property maps to Terraform and to the client.
type Kind string

// Attribute kinds.
const (
	KindString       Kind = "string"        // string scalar (also date, binary is skipped)
	KindInt          Kind = "int"           // integer
	KindFloat        Kind = "float"         // number
	KindBool         Kind = "bool"          // boolean
	KindDateTime     Kind = "datetime"      // string in TF, time.Time in the client
	KindChoice       Kind = "choice"        // string with a fixed set of values
	KindChoiceInt    Kind = "choice_int"    // integer with a fixed set of values
	KindFK           Kind = "fk"            // foreign key -> <name>_id Int64
	KindFKList       Kind = "fk_list"       // []int M2M -> <singular>_ids Set(Int64)
	KindIntList      Kind = "int_list"      // []int plain -> Set(Int64)
	KindStringList   Kind = "string_list"   // []string -> Set/List(String)
	KindTags         Kind = "tags"          // []NestedTagRequest -> Set(String) of slugs
	KindCustomFields Kind = "custom_fields" // free object -> jsontypes.Normalized
	// KindCustomFieldsJSON is the data-source form: a JSON string, because
	// dynamic values cannot be nested inside list data source items.
	KindCustomFieldsJSON Kind = "custom_fields_json"
	KindJSON             Kind = "json"           // untyped -> jsontypes.Normalized
	KindNestedList       Kind = "nested_list"    // []struct -> ListNestedAttribute
	KindIntRangeList     Kind = "int_range_list" // [][]int -> List(List(Int64))
	KindAnyList          Kind = "any_list"       // [][]any -> List(List(String))
)

// ReadKind classifies the shape of the same property on the read schema.
type ReadKind string

// Read kinds.
const (
	ReadAbsent     ReadKind = "absent"      // write-only property
	ReadScalar     ReadKind = "scalar"      // string/int/float/bool/time
	ReadChoice     ReadKind = "choice"      // {value,label} object (ChoiceString/ChoiceInteger)
	ReadBrief      ReadKind = "brief"       // nested object with an id
	ReadBriefList  ReadKind = "brief_list"  // []object with ids
	ReadTags       ReadKind = "tags"        // []NestedTag
	ReadMap        ReadKind = "map"         // map[string]any
	ReadJSON       ReadKind = "json"        // interface{}
	ReadNestedList ReadKind = "nested_list" // []struct
	ReadIntList    ReadKind = "int_list"    // []int
	ReadStringList ReadKind = "string_list" // []string
	ReadIntRange   ReadKind = "int_range"   // [][]int
	ReadAnyList    ReadKind = "any_list"    // [][]any
)

// Attr is one Terraform attribute and its mapping to the client models.
type Attr struct {
	// Name is the Terraform attribute name (snake_case).
	Name string
	// JSON is the API property name.
	JSON string
	// GoField is the field / setter suffix on the client request model.
	GoField string
	Kind    Kind

	Required bool
	// Nullable means the API accepts null (the client uses a Nullable* wrapper
	// for scalars when the property is optional or required+nullable).
	Nullable bool
	// Optional+Computed attribute (server-side default); ToAPI skips unknown/null.
	Computed bool
	// ReadOnly attributes exist only on the read schema (id, url, created, ...).
	ReadOnly        bool
	WriteOnly       bool
	Sensitive       bool
	RequiresReplace bool
	// DefaultEmptyString marks Optional+Computed strings that default to "".
	DefaultEmptyString bool
	// DefaultEmptySet marks set attributes that default to [] and are always sent.
	DefaultEmptySet bool
	// DefaultInt is the static default of an Optional+Computed integer or FK.
	DefaultInt *int64
	// OrderedList renders a List instead of a Set.
	OrderedList bool

	Description string
	Enum        []string
	EnumInts    []int64
	MaxLength   *int
	MinLength   *int
	Pattern     string
	// Precision is the number of decimals NetBox stores for floats (0 = unknown).
	Precision int

	// StringMap marks JSON attributes whose client type is map[string]string.
	StringMap bool
	// FoldCase marks strings NetBox normalises to upper case (MAC addresses, WWNs).
	FoldCase bool
	// KeepPriorWhenNull marks exposed read-only attributes that NetBox returns
	// only once (token secrets): the prior state is kept when the API returns null.
	KeepPriorWhenNull bool

	// Client typing.
	Int64    bool // client scalar is int64 (else int32)
	Float32  bool // client scalar is float32 (else float64)
	DateOnly bool // string with format date

	// Read side.
	ReadKind    ReadKind
	ReadGoField string
	// ReadChoiceInt marks ChoiceInteger read objects.
	ReadChoiceInt bool

	// FK target Terraform resource name ("" when unknown).
	Target string

	// Nested list item typing.
	Nested        []Attr
	NestedReqType string // client request item type, e.g. GenericObjectRequest
	NestedRspType string // client read item type, e.g. GenericObject
	// NestedItemRequired lists the required properties of the item request
	// schema in declaration order (constructor arguments).
	NestedItemRequired []string
}

// IsSet reports whether the attribute is a Set/List collection.
func (a Attr) IsCollection() bool {
	switch a.Kind {
	case KindFKList, KindIntList, KindStringList, KindTags, KindNestedList, KindIntRangeList, KindAnyList:
		return true
	}
	return false
}

// Filter is a list-endpoint query parameter usable by data sources.
type Filter struct {
	Name        string
	Description string
}

// Test holds acceptance-test configuration.
type Test struct {
	// Basic and Update are complete HCL configurations using {{.Name}} for a
	// unique name; the resource under test must be addressed as
	// netbox_<name>.test.
	Basic  string
	Update string
	// Auto is set when Basic/Update were synthesised by the generator.
	Auto bool
	// Skip disables the acceptance test with a reason.
	Skip string
	// Serial runs the test with resource.Test instead of ParallelTest.
	Serial bool
	// ImportVerifyIgnore lists attributes ignored by ImportStateVerify.
	ImportVerifyIgnore []string
	// Checks lists extra attribute checks: attribute -> expected string after Basic.
	Checks map[string]string
}

// Resource is one generated Terraform resource (plus its data sources).
type Resource struct {
	App         string // "dcim"
	PathSegment string // "sites"
	Path        string // "/api/dcim/sites/"
	Name        string // Terraform type without provider prefix: "site"
	Plural      string // list data source name: "sites"
	GoName      string // "Site" (Go identifier prefix in generated code)
	Description string

	ReadSchema   string // component name: Site
	CreateSchema string // WritableSiteRequest
	PatchSchema  string // PatchedWritableSiteRequest
	ReadType     string // Go type names
	CreateType   string
	PatchType    string
	Service      string // DcimAPI
	OpPrefix     string // DcimSites
	// CreateRequired lists required create properties in declaration order
	// (constructor argument order).
	CreateRequired []string

	Attrs         []Attr // attributes on the resource (write side + computed)
	ReadOnlyAttrs []Attr // extra computed attributes exposed by data sources only
	Filters       []Filter
	Lookups       []string // attribute names usable to look up a single object
	Deps          []string // Terraform names of FK targets
	Test          Test

	HasTags         bool
	HasCustomFields bool
	Skip            bool
	SkipReason      string
	// DataSourceOnly resources have no create (read-only collections).
	DataSourceOnly bool
}

// Attr returns the attribute with the given Terraform name.
func (r *Resource) Attr(name string) *Attr {
	for i := range r.Attrs {
		if r.Attrs[i].Name == name {
			return &r.Attrs[i]
		}
	}
	return nil
}

// TFType returns the full Terraform type name.
func (r *Resource) TFType() string { return "netbox_" + r.Name }

// TFPluralType returns the list data source type name.
func (r *Resource) TFPluralType() string { return "netbox_" + r.Plural }

// ObjectType returns the NetBox content type label ("dcim.site") of the
// resource, derived from the app and the read schema (Django model) name.
func (r *Resource) ObjectType() string { return r.App + "." + strings.ToLower(r.ReadType) }
