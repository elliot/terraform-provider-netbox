// Command gen generates the Terraform provider's resources, data sources,
// acceptance tests and examples from the NetBox OpenAPI document.
//
//	go run ./internal/gen                 # regenerate everything
//	go run ./internal/gen -dump           # print the resource model
//	go run ./internal/gen -only site,vrf  # limit to some resources
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/elliot/terraform-provider-netbox/internal/gen/build"
	"github.com/elliot/terraform-provider-netbox/internal/gen/model"
	"github.com/elliot/terraform-provider-netbox/internal/gen/openapi"
	"github.com/elliot/terraform-provider-netbox/internal/gen/overrides"
	"github.com/elliot/terraform-provider-netbox/internal/gen/render"
)

func main() {
	var (
		specPath  = flag.String("spec", "", "OpenAPI document (default spec/netbox-<VERSION>.openapi.json)")
		ovDir     = flag.String("overrides", "generator/overrides", "directory of overrides YAML files")
		outDir    = flag.String("out", "internal/provider/gen", "output directory for generated Go packages")
		exDir     = flag.String("examples", "examples", "output directory for examples")
		docsDir   = flag.String("docs", "docs", "output directory for generated markdown tables")
		tmplDir   = flag.String("templates", "templates", "output directory for tfplugindocs templates")
		only      = flag.String("only", "", "comma-separated resource names to generate")
		dump      = flag.Bool("dump", false, "print the resource model and exit")
		listNames = flag.Bool("list", false, "print resource names and exit")
	)
	flag.Parse()
	if *specPath == "" {
		v, err := os.ReadFile("spec/VERSION")
		if err != nil {
			fatal(err)
		}
		*specPath = fmt.Sprintf("spec/netbox-%s.openapi.json", strings.TrimSpace(string(v)))
	}
	doc, err := openapi.Load(*specPath)
	if err != nil {
		fatal(err)
	}
	ov, err := overrides.Load(*ovDir)
	if err != nil {
		fatal(err)
	}
	resources, warnings, err := build.Build(doc, ov)
	if err != nil {
		fatal(err)
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	if *only != "" {
		want := map[string]bool{}
		for _, n := range strings.Split(*only, ",") {
			want[strings.TrimSpace(n)] = true
		}
		var sel []*model.Resource
		for _, r := range resources {
			if want[r.Name] {
				sel = append(sel, r)
			}
		}
		resources = sel
	}
	switch {
	case *listNames:
		for _, r := range resources {
			flag := ""
			if r.Skip {
				flag = " (skipped: " + r.SkipReason + ")"
			} else if r.DataSourceOnly {
				flag = " (data source only)"
			}
			fmt.Printf("%-16s %-40s netbox_%s / netbox_%s%s\n", r.App, r.Path, r.Name, r.Plural, flag)
		}
	case *dump:
		dumpModel(resources)
	default:
		if err := render.All(resources, doc.Info.Version, render.Options{OutDir: *outDir, ExamplesDir: *exDir, DocsDir: *docsDir, TemplatesDir: *tmplDir, Partial: *only != ""}); err != nil {
			fatal(err)
		}
	}
}

func dumpModel(resources []*model.Resource) {
	kinds := map[model.Kind]int{}
	for _, r := range resources {
		fmt.Printf("== %s  netbox_%s (%s.%s) create=%s patch=%s read=%s deps=%v lookups=%v tags=%v cf=%v skip=%v\n",
			r.Path, r.Name, r.Service, r.OpPrefix, r.CreateType, r.PatchType, r.ReadType, r.Deps, r.Lookups, r.HasTags, r.HasCustomFields, r.Skip)
		for _, a := range r.Attrs {
			kinds[a.Kind]++
			printAttr("  ", a)
		}
		if len(r.ReadOnlyAttrs) > 0 {
			names := make([]string, 0, len(r.ReadOnlyAttrs))
			for _, a := range r.ReadOnlyAttrs {
				names = append(names, a.Name+":"+string(a.Kind))
			}
			fmt.Printf("  [ds-only] %s\n", strings.Join(names, " "))
		}
		fmt.Printf("  [filters] %d\n", len(r.Filters))
	}
	var ks []string
	for k, n := range kinds {
		ks = append(ks, fmt.Sprintf("%s=%d", k, n))
	}
	sort.Strings(ks)
	fmt.Println("kinds:", strings.Join(ks, " "))
}

func printAttr(indent string, a model.Attr) {
	flags := []string{}
	if a.Required {
		flags = append(flags, "required")
	}
	if a.Computed {
		flags = append(flags, "computed")
	}
	if a.Nullable {
		flags = append(flags, "nullable")
	}
	if a.ReadOnly {
		flags = append(flags, "ro")
	}
	if a.DefaultEmptyString {
		flags = append(flags, "def=''")
	}
	if a.DefaultEmptySet {
		flags = append(flags, "def=[]")
	}
	if a.Target != "" {
		flags = append(flags, "->"+a.Target)
	}
	if len(a.Enum) > 0 {
		flags = append(flags, fmt.Sprintf("enum%d", len(a.Enum)))
	}
	fmt.Printf("%s%-28s %-14s read=%-12s go=%s %s\n", indent, a.Name, a.Kind, a.ReadKind, a.GoField, strings.Join(flags, ","))
	for _, n := range a.Nested {
		printAttr(indent+"    ", n)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "gen:", err)
	os.Exit(1)
}
