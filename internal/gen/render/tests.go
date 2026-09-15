package render

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/elliot/terraform-provider-netbox/internal/gen/model"
	"github.com/elliot/terraform-provider-netbox/internal/gen/naming"
)

// fixtureAttrs returns HCL for the required attributes that can be synthesised
// without other objects, or ok=false when the resource needs dependencies.
func autoFixture(r *model.Resource, update bool) (string, bool) {
	var lines []string
	for _, a := range r.Attrs {
		if !a.Required || a.ReadOnly {
			continue
		}
		switch a.Kind {
		case model.KindString:
			if a.DateOnly {
				lines = append(lines, fmt.Sprintf("  %s = %q", a.Name, "2024-01-01"))
			} else if a.Name == "slug" {
				lines = append(lines, fmt.Sprintf("  %s = \"{{.Name}}\"", a.Name))
			} else if a.MaxLength != nil && *a.MaxLength < 20 {
				lines = append(lines, fmt.Sprintf("  %s = %q", a.Name, "tfacc"))
			} else {
				lines = append(lines, fmt.Sprintf("  %s = \"{{.Name}}\"", a.Name))
			}
		case model.KindChoice:
			if len(a.Enum) == 0 {
				return "", false
			}
			lines = append(lines, fmt.Sprintf("  %s = %q", a.Name, a.Enum[0]))
		case model.KindChoiceInt:
			if len(a.EnumInts) == 0 {
				return "", false
			}
			lines = append(lines, fmt.Sprintf("  %s = %d", a.Name, a.EnumInts[0]))
		case model.KindInt:
			lines = append(lines, fmt.Sprintf("  %s = 1", a.Name))
		case model.KindFloat:
			lines = append(lines, fmt.Sprintf("  %s = 1", a.Name))
		case model.KindBool:
			lines = append(lines, fmt.Sprintf("  %s = true", a.Name))
		default:
			return "", false
		}
	}
	if update {
		if d := r.Attr("description"); d != nil {
			lines = append(lines, "  description = \"updated by acceptance test\"")
		} else if c := r.Attr("comments"); c != nil {
			lines = append(lines, "  comments = \"updated by acceptance test\"")
		} else {
			return "", false
		}
	} else if d := r.Attr("description"); d != nil {
		lines = append(lines, "  description = \"created by acceptance test\"")
	}
	if r.HasTags && !update {
		// exercise tags with a dedicated tag resource
		lines = append(lines, "  tags = [netbox_tag.test.slug]")
	}
	var b strings.Builder
	if r.HasTags && !update {
		b.WriteString("resource \"netbox_tag\" \"test\" {\n  name = \"{{.Name}}-tag\"\n  slug = \"{{.Name}}-tag\"\n}\n\n")
	}
	fmt.Fprintf(&b, "resource %q \"test\" {\n%s\n}\n", r.TFType(), strings.Join(lines, "\n"))
	return b.String(), true
}

// testFile renders <name>_resource_test.go.
func testFile(r *model.Resource, version string) string {
	basic, update := r.Test.Basic, r.Test.Update
	auto := false
	if basic == "" {
		if f, ok := autoFixture(r, false); ok {
			basic, auto = f, true
		}
	}
	if update == "" && auto {
		if f, ok := autoFixture(r, true); ok {
			update = f
		}
	}
	skip := r.Test.Skip
	if skip == "" && basic == "" {
		skip = fmt.Sprintf("no acceptance fixture: add test.basic/test.update for %q in generator/overrides/%s.yaml", r.PathSegment, r.App)
	}
	ignore := append([]string{}, r.Test.ImportVerifyIgnore...)
	if r.HasCustomFields {
		ignore = append(ignore, "custom_fields")
	}
	for _, a := range r.Attrs {
		if a.WriteOnly || a.Sensitive {
			ignore = append(ignore, a.Name)
		}
	}
	sort.Strings(ignore)

	var b strings.Builder
	fmt.Fprintf(&b, generatedHeader, version)
	fmt.Fprintf(&b, "package %s_test\n\n", pkgName(r.App))
	b.WriteString(`import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/elliot/terraform-provider-netbox/internal/acctest"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
)

`)
	fmt.Fprintf(&b, "const %sTestConfigBasic = %s\n\n", lower(r.GoName), backtick(basic))
	fmt.Fprintf(&b, "const %sTestConfigUpdate = %s\n\n", lower(r.GoName), backtick(update))
	dsCfg := fmt.Sprintf(`
data %[1]q "by_id" {
  id = %[1]s.test.id
}

data %[2]q "list" {
  filters = [{ name = "id", value = tostring(%[1]s.test.id) }]
}
`, r.TFType(), r.TFPluralType())
	fmt.Fprintf(&b, "const %sTestConfigDataSources = %s\n\n", lower(r.GoName), backtick(dsCfg))

	testFunc := "resource.ParallelTest"
	if r.Test.Serial {
		testFunc = "resource.Test"
	}
	checks := ""
	for _, k := range sortedKeys(r.Test.Checks) {
		checks += fmt.Sprintf("\t\t\t\t\tresource.TestCheckResourceAttr(%q, %q, %q),\n", r.TFType()+".test", k, r.Test.Checks[k])
	}
	fmt.Fprintf(&b, `func TestAcc%[1]s_basic(t *testing.T) {
`, r.GoName)
	if skip != "" {
		fmt.Fprintf(&b, "\tt.Skip(%q)\n", skip)
	}
	fmt.Fprintf(&b, `	name := acctest.RandName()
	steps := []resource.TestStep{
		{
			Config: acctest.Render(t, %[2]sTestConfigBasic, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet(%[3]q, "id"),
%[4]s			),
		},
`, r.GoName, lower(r.GoName), r.TFType()+".test", checks)
	if update != "" {
		fmt.Fprintf(&b, `		{
			Config: acctest.Render(t, %[1]sTestConfigUpdate, name),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(%[2]q, plancheck.ResourceActionUpdate)},
			},
		},
`, lower(r.GoName), r.TFType()+".test")
	}
	lastCfg := lower(r.GoName) + "TestConfigBasic"
	if update != "" {
		lastCfg = lower(r.GoName) + "TestConfigUpdate"
	}
	// Data source step: read the object back by id and through the list data source.
	fmt.Fprintf(&b, `		{
			Config: acctest.Render(t, %[1]s+%[2]sTestConfigDataSources, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrPair("data.%[3]s.by_id", "id", %[4]q, "id"),
				resource.TestCheckResourceAttr("data.%[5]s.list", "items.#", "1"),
				resource.TestCheckResourceAttrPair("data.%[5]s.list", "items.0.id", %[4]q, "id"),
			),
		},
`, lastCfg, lower(r.GoName), r.TFType(), r.TFType()+".test", r.TFPluralType())
	ign := ""
	if len(ignore) > 0 {
		ign = "\t\t\tImportStateVerifyIgnore: []string{\"" + strings.Join(ignore, "\", \"") + "\"},\n"
	}
	fmt.Fprintf(&b, `		{
			ResourceName:      %[1]q,
			ImportState:       true,
			ImportStateVerify: true,
%[2]s		},
		{
			Config: acctest.Render(t, %[3]s, name),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
		},
	}
	%[4]s(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:             acctest.CheckDestroyed(%[5]q, %[6]q),
		Steps:                    steps,
	})
}

func init() {
	resource.AddTestSweepers(%[5]q, &resource.Sweeper{
		Name:         %[5]q,
		Dependencies: %[7]s,
		F: func(_ string) error {
			return acctest.Sweep(%[6]q, %[8]s)
		},
	})
}
`, r.TFType()+".test", ign, lastCfg, testFunc, r.TFType(), r.Path, goStringSlice(depTypes(r)), goStringSlice(sweepFilters(r)))
	return b.String()
}

// depTypes returns the Terraform types that must be swept before this one
// (objects referencing this resource), which is the inverse of r.Deps; the
// sweeper framework runs dependencies first, so we list resources that depend
// on us. That information is not local, so we list nothing and rely on the
// sweeper being retried; instead we express our own deps as "run after".
func depTypes(r *model.Resource) []string {
	// resource.Sweeper.Dependencies are sweepers that must run BEFORE this one.
	// Our FK targets must be swept AFTER us, so they are not dependencies here;
	// dependents are computed globally in render.All (see sweeperDeps).
	return sweeperDeps[r.TFType()]
}

// sweeperDeps maps a resource type to the types that must be swept before it.
var sweeperDeps = map[string][]string{}

// computeSweeperDeps inverts the FK graph: if A references B, A is swept before B.
func computeSweeperDeps(resources []*model.Resource) {
	byName := map[string]*model.Resource{}
	for _, r := range resources {
		byName[r.Name] = r
	}
	for _, r := range resources {
		if r.Skip || r.DataSourceOnly {
			continue
		}
		for _, d := range r.Deps {
			if t, ok := byName[d]; ok && !t.Skip && !t.DataSourceOnly && t.Name != r.Name {
				sweeperDeps[t.TFType()] = append(sweeperDeps[t.TFType()], r.TFType())
			}
		}
	}
	for k := range sweeperDeps {
		sort.Strings(sweeperDeps[k])
	}
}

func sweepFilters(r *model.Resource) []string {
	have := map[string]bool{}
	for _, f := range r.Filters {
		have[f.Name] = true
	}
	var out []string
	for _, f := range []string{"name__isw", "slug__isw", "description__isw", "cid__isw", "username__isw", "q"} {
		if have[f] {
			out = append(out, f)
		}
	}
	return out
}

func goStringSlice(items []string) string {
	if len(items) == 0 {
		return "[]string{}"
	}
	return "[]string{\"" + strings.Join(items, "\", \"") + "\"}"
}

func backtick(s string) string {
	if strings.Contains(s, "`") {
		return q(s)
	}
	return "`" + s + "`"
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---- examples ----

func writeExamples(r *model.Resource, dir string) error {
	d := filepath.Join(dir, "resources", r.TFType())
	if err := os.MkdirAll(d, 0o750); err != nil {
		return err
	}
	example := r.Test.Basic
	if example == "" {
		if f, ok := autoFixture(r, false); ok {
			example = f
		} else {
			example = skeletonExample(r)
		}
	}
	example = strings.ReplaceAll(example, "{{.Name}}", "example")
	example = strings.ReplaceAll(example, "\"test\"", "\"example\"")
	example = strings.ReplaceAll(example, ".test.", ".example.")
	example = strings.ReplaceAll(example, "created by acceptance test", "Managed by Terraform")
	if err := writeIfGenerated(filepath.Join(d, "resource.tf"), example); err != nil {
		return err
	}
	imp := fmt.Sprintf("# %s can be imported by its NetBox object ID.\nterraform import %s.example 123\n", r.TFType(), r.TFType())
	return writeIfGenerated(filepath.Join(d, "import.sh"), imp)
}

func writeDataSourceExamples(r *model.Resource, dir string) error {
	d := filepath.Join(dir, "data-sources", r.TFType())
	if err := os.MkdirAll(d, 0o750); err != nil {
		return err
	}
	var single string
	switch {
	case len(r.Lookups) > 0:
		a := r.Attr(r.Lookups[0])
		val := "\"example\""
		if a != nil && tfType(*a) == "types.Int64" {
			val = "65000"
		}
		single = fmt.Sprintf("data %q \"example\" {\n  %s = %s\n}\n", r.TFType(), r.Lookups[0], val)
	default:
		single = fmt.Sprintf("data %q \"example\" {\n  id = 123\n}\n", r.TFType())
	}
	single += fmt.Sprintf("\n# Lookup by arbitrary API filters:\ndata %q \"filtered\" {\n  filters = [\n    { name = \"id\", value = \"123\" },\n  ]\n}\n", r.TFType())
	if err := writeIfGenerated(filepath.Join(d, "data-source.tf"), single); err != nil {
		return err
	}
	dp := filepath.Join(dir, "data-sources", r.TFPluralType())
	if err := os.MkdirAll(dp, 0o750); err != nil {
		return err
	}
	list := fmt.Sprintf("data %q \"all\" {}\n\ndata %q \"filtered\" {\n  filters = [\n    { name = \"q\", value = \"example\" },\n  ]\n  limit = 10\n}\n\noutput \"%s_ids\" {\n  value = data.%s.filtered.items[*].id\n}\n", r.TFPluralType(), r.TFPluralType(), r.Plural, r.TFPluralType())
	return writeIfGenerated(filepath.Join(dp, "data-source.tf"), list)
}

// writeIfGenerated writes an example unless a hand-edited one exists (a file
// without the generated marker comment is left alone).
func writeIfGenerated(path, content string) error {
	marker := "# Generated by internal/gen; edit generator/overrides to change, or remove this line to hand-maintain.\n"
	if data, err := os.ReadFile(path); err == nil && !strings.HasPrefix(string(data), marker) {
		return nil
	}
	return os.WriteFile(path, []byte(marker+content), 0o600)
}

func skeletonExample(r *model.Resource) string {
	var lines []string
	for _, a := range r.Attrs {
		if !a.Required || a.ReadOnly {
			continue
		}
		switch a.Kind {
		case model.KindFK:
			lines = append(lines, fmt.Sprintf("  %s = netbox_%s.example.id", a.Name, orDefault(a.Target, naming.Singular(strings.TrimSuffix(a.Name, "_id")))))
		case model.KindString, model.KindChoice:
			v := "example"
			if len(a.Enum) > 0 {
				v = a.Enum[0]
			}
			lines = append(lines, fmt.Sprintf("  %s = %q", a.Name, v))
		case model.KindInt, model.KindChoiceInt:
			lines = append(lines, fmt.Sprintf("  %s = 1", a.Name))
		case model.KindBool:
			lines = append(lines, fmt.Sprintf("  %s = true", a.Name))
		case model.KindNestedList:
			lines = append(lines, fmt.Sprintf("  %s = [\n    # { ... }\n  ]", a.Name))
		case model.KindJSON:
			lines = append(lines, fmt.Sprintf("  %s = jsonencode({})", a.Name))
		default:
			lines = append(lines, fmt.Sprintf("  %s = []", a.Name))
		}
	}
	return fmt.Sprintf("resource %q \"example\" {\n%s\n}\n", r.TFType(), strings.Join(lines, "\n"))
}

func orDefault(s, d string) string {
	if s != "" {
		return s
	}
	return d
}
