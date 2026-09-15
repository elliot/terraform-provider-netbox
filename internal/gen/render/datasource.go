package render

import (
	"fmt"
	"strings"

	"github.com/elliot/terraform-provider-netbox/internal/gen/model"
	"github.com/elliot/terraform-provider-netbox/internal/gen/naming"
)

// dataSourceFile renders <name>_data_source.go: the singular data source
// (lookup by id, lookup attributes or filters) and the plural list data source.
func dataSourceFile(r *model.Resource, version string) string {
	var b strings.Builder
	fmt.Fprintf(&b, generatedHeader, version)
	fmt.Fprintf(&b, "package %s\n\n", pkgName(r.App))
	b.WriteString(commonImports)

	g, lg := r.GoName, lower(r.GoName)
	all := append(append([]model.Attr{}, r.Attrs...), r.ReadOnlyAttrs...)
	for i := range all {
		if all[i].Kind == model.KindCustomFields {
			all[i].Kind = model.KindCustomFieldsJSON
			all[i].Description = "All custom field values of the object as a JSON object (`jsondecode(...)`); selection values are unwrapped to the choice value and related objects to their ID."
		}
	}
	lookups := map[string]bool{"id": true}
	for _, l := range r.Lookups {
		lookups[l] = true
	}

	fmt.Fprintf(&b, "func init() {\nprovider.RegisterDataSource(New%[1]sDataSource)\nprovider.RegisterDataSource(New%[1]sListDataSource)\n}\n\n", g)
	if r.DataSourceOnly {
		b.WriteString(enumVars(r))
		b.WriteString(nestedTypes(r, r.Attrs))
	}
	fmt.Fprintf(&b, "// %sDataModel is the state of data.%s and of items of data.%s.\n", g, r.TFType(), r.TFPluralType())
	b.WriteString(modelStruct(g+"DataModel", all))
	fmt.Fprintf(&b, "// %sFilterNames lists the query parameters accepted by %s (sorted).\nvar %sFilterNames = []string{\n", lg, r.Path, lg)
	for i, n := range sortedFilterNames(r) {
		b.WriteString(q(n) + ",")
		if i%6 == 5 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n}\n\n")

	// attributes
	fmt.Fprintf(&b, "// %sDataAttributes returns the attributes shared by the singular and list data sources.\nfunc %sDataAttributes(lookup bool) map[string]dsschema.Attribute {\nattrs := map[string]dsschema.Attribute{\n", lg, lg)
	for _, a := range all {
		b.WriteString(schemaAttr(r, a, "dsschema", true, false, ""))
	}
	b.WriteString("}\nif lookup {\n")
	for _, a := range all {
		if lookups[a.Name] {
			b.WriteString(schemaAttrAssign(r, a))
		}
	}
	b.WriteString("}\nreturn attrs\n}\n\n")

	filtersDesc := q("Additional query filters as name/value pairs, e.g. `{ name = \"tenant_id\", value = \"12\" }`. Any query parameter of `" + r.Path + "` is accepted.")
	filtersAttr := `"filters": dsschema.ListNestedAttribute{
		MarkdownDescription: ` + filtersDesc + `,
		Optional: true,
		NestedObject: dsschema.NestedAttributeObject{Attributes: map[string]dsschema.Attribute{
			"name":  dsschema.StringAttribute{Required: true, MarkdownDescription: "Query parameter name."},
			"value": dsschema.StringAttribute{Required: true, MarkdownDescription: "Query parameter value."},
		}},
	},
`
	lookupDoc := "`id`"
	for _, l := range r.Lookups {
		lookupDoc += ", `" + l + "`"
	}

	// Singular data source.
	fmt.Fprintf(&b, `var (
	_ datasource.DataSource              = &%[1]sDataSource{}
	_ datasource.DataSourceWithConfigure = &%[1]sDataSource{}
)

// %[1]sDataSource reads one %[2]s.
type %[1]sDataSource struct {
	client *netbox.APIClient
}

// %[1]sDataSourceModel is the state of data.%[2]s.
type %[1]sDataSourceModel struct {
	%[1]sDataModel
	Filters types.List `+"`tfsdk:\"filters\"`"+`
}

// New%[1]sDataSource returns a new %[2]s data source.
func New%[1]sDataSource() datasource.DataSource { return &%[1]sDataSource{} }

func (d *%[1]sDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + %[3]q
}

func (d *%[1]sDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := %[4]sDataAttributes(true)
	%[5]s
	resp.Schema = dsschema.Schema{
		MarkdownDescription: %[6]s,
		Attributes: attrs,
	}
}

func (d *%[1]sDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*provider.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *provider.ProviderData, got %%T", req.ProviderData))
		return
	}
	d.client = pd.API
}

func (d *%[1]sDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg %[1]sDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var obj *netbox.%[7]s
	if conv.Known(cfg.Id) {
		id, err := netbox.Int32ID(cfg.Id.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Invalid ID", err.Error())
			return
		}
		var out netbox.%[7]s
		if err := d.client.GetRaw(ctx, fmt.Sprintf("%[8]s%%d/", id), &out); err != nil {
			resp.Diagnostics.AddError("Error reading %[2]s", err.Error())
			return
		}
		obj = &out
	} else {
		query := url.Values{}
%[9]s
		conv.ApplyFilters(ctx, cfg.Filters, %[4]sFilterNames, query, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(query) == 0 {
			resp.Diagnostics.AddError("Missing lookup", "Set %[10]s or filters to select the object.")
			return
		}
		query.Set("limit", "2")
		var page netbox.Paginated%[7]sList
		if err := d.client.GetRaw(ctx, "%[8]s?"+query.Encode(), &page); err != nil {
			resp.Diagnostics.AddError("Error listing %[2]s", err.Error())
			return
		}
		switch n := page.GetCount(); {
		case n == 0:
			resp.Diagnostics.AddError("Not found", "No %[2]s matches the given lookup ("+query.Encode()+").")
			return
		case n > 1:
			resp.Diagnostics.AddError("Ambiguous lookup", fmt.Sprintf("%%d %[2]s objects match the given lookup (%%s); refine the filters.", n, query.Encode()))
			return
		}
		obj = &page.Results[0]
	}
	%[4]sDataFromAPI(ctx, obj, &cfg.%[1]sDataModel, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}

`, g, r.TFType(), "_"+r.Name, lg, "attrs[\"filters\"] = "+strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(filtersAttr), "\"filters\":"), ",")),
		q(fmt.Sprintf("Reads a single NetBox %s (`%s`) by %s or filters.", humanName(r), r.Path, lookupDoc)), r.ReadType, r.Path, lookupCode(r), lookupDoc)

	// List data source.
	fmt.Fprintf(&b, `var (
	_ datasource.DataSource              = &%[1]sListDataSource{}
	_ datasource.DataSourceWithConfigure = &%[1]sListDataSource{}
)

// %[1]sListDataSource lists %[2]s objects.
type %[1]sListDataSource struct {
	client *netbox.APIClient
}

// %[1]sListDataSourceModel is the state of data.%[2]s.
type %[1]sListDataSourceModel struct {
	Filters types.List  `+"`tfsdk:\"filters\"`"+`
	Limit   types.Int64 `+"`tfsdk:\"limit\"`"+`
	Items   types.List  `+"`tfsdk:\"items\"`"+`
}

// New%[1]sListDataSource returns a new %[2]s data source.
func New%[1]sListDataSource() datasource.DataSource { return &%[1]sListDataSource{} }

func (d *%[1]sListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + %[3]q
}

func (d *%[1]sListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: %[4]s,
		Attributes: map[string]dsschema.Attribute{
%[5]s
			"limit": dsschema.Int64Attribute{
				MarkdownDescription: "Maximum number of objects to return (default: all).",
				Optional: true,
			},
			"items": dsschema.ListNestedAttribute{
				MarkdownDescription: "The matching objects.",
				Computed: true,
				NestedObject: dsschema.NestedAttributeObject{Attributes: %[6]sDataAttributes(false)},
			},
		},
	}
}

func (d *%[1]sListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*provider.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *provider.ProviderData, got %%T", req.ProviderData))
		return
	}
	d.client = pd.API
}

func (d *%[1]sListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg %[1]sListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	query := url.Values{}
	conv.ApplyFilters(ctx, cfg.Filters, %[6]sFilterNames, query, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	limit := int64(-1)
	if conv.Known(cfg.Limit) {
		limit = cfg.Limit.ValueInt64()
	}
	pageSize := int64(netbox.DefaultPageSize)
	if limit >= 0 && limit < pageSize {
		pageSize = limit
	}
	var objs []netbox.%[7]s
	offset := int64(0)
	for limit != 0 {
		query.Set("limit", strconv.FormatInt(pageSize, 10))
		query.Set("offset", strconv.FormatInt(offset, 10))
		var page netbox.Paginated%[7]sList
		if err := d.client.GetRaw(ctx, "%[8]s?"+query.Encode(), &page); err != nil {
			resp.Diagnostics.AddError("Error listing %[2]s", err.Error())
			return
		}
		objs = append(objs, page.Results...)
		offset += int64(len(page.Results))
		if len(page.Results) == 0 || offset >= int64(page.GetCount()) || (limit > 0 && int64(len(objs)) >= limit) {
			break
		}
	}
	if limit > 0 && int64(len(objs)) > limit {
		objs = objs[:limit]
	}
	items := make([]%[1]sDataModel, 0, len(objs))
	for i := range objs {
		var m %[1]sDataModel
		%[6]sDataFromAPI(ctx, &objs[i], &m, &resp.Diagnostics)
		items = append(items, m)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: provider.AttrTypes(%[6]sDataAttributes(false))}, items)
	resp.Diagnostics.Append(diags...)
	cfg.Items = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}

`, g, r.TFPluralType(), "_"+r.Plural, q(fmt.Sprintf("Lists NetBox %s objects (`%s`) matching the given filters.", humanName(r), r.Path)), filtersAttr, lg, r.ReadType, r.Path)

	b.WriteString(fromAPIFunc(r, true))
	return b.String()
}

// schemaAttrAssign re-renders a lookup attribute as an assignment into attrs.
func schemaAttrAssign(r *model.Resource, a model.Attr) string {
	s := schemaAttr(r, a, "dsschema", true, true, "")
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ",")
	i := strings.Index(s, ":")
	return fmt.Sprintf("attrs[%s] = %s\n", s[:i], strings.TrimSpace(s[i+1:]))
}

// lookupCode renders the query construction for the singular data source lookups.
func lookupCode(r *model.Resource) string {
	var b strings.Builder
	for _, l := range r.Lookups {
		a := r.Attr(l)
		if a == nil {
			continue
		}
		f := "cfg." + field(*a)
		var expr string
		switch tfType(*a) {
		case "types.String":
			expr = f + ".ValueString()"
		case "types.Int64":
			expr = "strconv.FormatInt(" + f + ".ValueInt64(), 10)"
		case "types.Bool":
			expr = "strconv.FormatBool(" + f + ".ValueBool())"
		case "types.Float64":
			expr = "strconv.FormatFloat(" + f + ".ValueFloat64(), 'f', -1, 64)"
		default:
			continue
		}
		fmt.Fprintf(&b, "\t\tif conv.Known(%s) {\n\t\t\tquery.Set(%q, %s)\n\t\t}\n", f, l, expr)
	}
	return b.String()
}

func humanName(r *model.Resource) string {
	return strings.ReplaceAll(naming.Singular(r.Name), "_", " ")
}
