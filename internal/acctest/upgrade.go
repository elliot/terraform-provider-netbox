package acctest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// Environment variables of the upgrade tests.
const (
	// EnvUpgradeFrom selects the released provider version the upgrade tests
	// start from (a Terraform version constraint, normally an exact version
	// such as 0.5.0). Upgrade tests are skipped when it is unset.
	EnvUpgradeFrom = "NETBOX_UPGRADE_FROM"
	// EnvUpgradeSource overrides the registry source of that release
	// (default DefaultUpgradeSource), e.g. to test from a fork.
	EnvUpgradeSource = "NETBOX_UPGRADE_SOURCE"
)

// DefaultUpgradeSource is the registry address of the published provider.
const DefaultUpgradeSource = "elliot/netbox"

// UpgradeCase describes one upgrade test: apply a configuration with a
// released provider from the registry, then expect the current build to plan
// no changes against the state that release wrote.
type UpgradeCase struct {
	// ResourceType is the Terraform type under test (for CheckDestroy).
	ResourceType string
	// APIPath is the NetBox collection path of ResourceType (for CheckDestroy).
	APIPath string
	// Configs are rendered configurations in order of preference. The first
	// one the released provider's schema accepts is used, so a fixture that
	// uses attributes newer than the release falls back to a smaller one.
	Configs []string
	// Serial runs the test with resource.Test instead of resource.ParallelTest.
	Serial bool
}

// UpgradeTest runs an UpgradeCase. It skips unless TF_ACC and
// NETBOX_UPGRADE_FROM are set, and when no configuration only uses resource
// types and attributes the released provider knows.
func UpgradeTest(t *testing.T, c UpgradeCase) {
	t.Helper()
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("acceptance tests skipped unless env %q is set", resource.EnvTfAcc)
	}
	from := os.Getenv(EnvUpgradeFrom)
	if from == "" {
		t.Skipf("upgrade tests skipped unless env %q is set to a released provider version", EnvUpgradeFrom)
	}
	PreCheck(t)
	source := os.Getenv(EnvUpgradeSource)
	if source == "" {
		source = DefaultUpgradeSource
	}
	schema, err := releasedSchema(source, from)
	if err != nil {
		t.Fatalf("read the schema of %s %s: %v", source, from, err)
	}
	var cfg string
	var reasons []string
	for i, candidate := range c.Configs {
		problems, err := incompatibilities(candidate, schema)
		if err != nil {
			t.Fatalf("config %d: %v", i, err)
		}
		if len(problems) == 0 {
			cfg = candidate
			break
		}
		reasons = append(reasons, fmt.Sprintf("config %d: %s", i, strings.Join(problems, ", ")))
	}
	if cfg == "" {
		t.Skipf("no configuration is compatible with %s %s (%s); add test.upgrade.config or test.upgrade.skip in generator/overrides if this persists",
			source, from, strings.Join(reasons, "; "))
	}

	tc := resource.TestCase{
		PreCheck:     func() { PreCheck(t) },
		CheckDestroy: CheckDestroyed(c.ResourceType, c.APIPath),
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"netbox": {Source: source, VersionConstraint: from},
				},
				Config: cfg,
			},
			{
				ProtoV6ProviderFactories: ProviderFactories,
				Config:                   cfg,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	}
	if c.Serial {
		resource.Test(t, tc)
	} else {
		resource.ParallelTest(t, tc)
	}
}

type schemaResult struct {
	schema *tfjson.ProviderSchema
	err    error
}

var (
	schemaMu    sync.Mutex
	schemaCache = map[string]schemaResult{}
)

// releasedSchema returns the schema of a released provider, fetched once per
// test binary with `terraform providers schema -json`. It also points
// TF_PLUGIN_CACHE_DIR at a persistent directory when unset, so the release is
// downloaded once instead of by every test's `terraform init`. It runs in the
// sequential part of the tests, before any parallel test resumes, so the
// cache is complete before parallel inits read it.
func releasedSchema(source, version string) (*tfjson.ProviderSchema, error) {
	schemaMu.Lock()
	defer schemaMu.Unlock()
	key := source + "@" + version
	if r, ok := schemaCache[key]; ok {
		return r.schema, r.err
	}
	s, err := fetchSchema(source, version)
	schemaCache[key] = schemaResult{s, err}
	return s, err
}

func fetchSchema(source, version string) (*tfjson.ProviderSchema, error) {
	bin := os.Getenv("TF_ACC_TERRAFORM_PATH")
	if bin == "" {
		p, err := exec.LookPath("terraform")
		if err != nil {
			return nil, fmt.Errorf("set TF_ACC_TERRAFORM_PATH or put terraform on PATH: %w", err)
		}
		bin = p
	}
	if os.Getenv("TF_PLUGIN_CACHE_DIR") == "" {
		if base, err := os.UserCacheDir(); err == nil {
			dir := filepath.Join(base, "terraform-provider-netbox", "plugin-cache")
			if err := os.MkdirAll(dir, 0o750); err == nil {
				_ = os.Setenv("TF_PLUGIN_CACHE_DIR", dir)
			}
		}
	}
	if os.Getenv("TF_PLUGIN_CACHE_DIR") != "" && os.Getenv("TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE") == "" {
		// Test working directories have no lock file, so without this every
		// init downloads the release again and rewrites the cache entry
		// while parallel tests link it, which fails their checksum check.
		_ = os.Setenv("TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE", "1")
	}
	dir, err := os.MkdirTemp("", "netbox-upgrade-schema-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	mainTF := fmt.Sprintf("terraform {\n  required_providers {\n    netbox = {\n      source  = %q\n      version = %q\n    }\n  }\n}\n", source, version)
	//nolint:gosec // dir comes from os.MkdirTemp
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(mainTF), 0o600); err != nil {
		return nil, err
	}
	run := func(args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, args...) //nolint:gosec // the Terraform binary chosen by the test environment
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "CHECKPOINT_DISABLE=1", "TF_IN_AUTOMATION=1")
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("terraform %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return stdout.Bytes(), nil
	}
	if _, err := run("init", "-input=false", "-backend=false", "-no-color"); err != nil {
		return nil, err
	}
	out, err := run("providers", "schema", "-json")
	if err != nil {
		return nil, err
	}
	var all tfjson.ProviderSchemas
	if err := json.Unmarshal(out, &all); err != nil {
		return nil, fmt.Errorf("decode provider schemas: %w", err)
	}
	for addr, s := range all.Schemas {
		if strings.HasSuffix(addr, "/"+source) {
			return s, nil
		}
	}
	return nil, fmt.Errorf("provider %s missing from `terraform providers schema` output", source)
}

// incompatibilities lists the resource types, data sources and attributes a
// configuration uses that the given provider schema lacks.
func incompatibilities(cfg string, schema *tfjson.ProviderSchema) ([]string, error) {
	file, diags := hclsyntax.ParseConfig([]byte(cfg), "config.tf", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse config: %s", diags.Error())
	}
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected HCL body type %T", file.Body)
	}
	var out []string
	for _, b := range body.Blocks {
		if len(b.Labels) == 0 {
			continue
		}
		var s *tfjson.Schema
		switch b.Type {
		case "resource":
			s = schema.ResourceSchemas[b.Labels[0]]
		case "data":
			s = schema.DataSourceSchemas[b.Labels[0]]
		default:
			continue
		}
		if s == nil || s.Block == nil {
			out = append(out, fmt.Sprintf("%s %s is unknown", b.Type, b.Labels[0]))
			continue
		}
		out = append(out, unknownAttrs(b.Labels[0], b.Body, s.Block)...)
	}
	return out, nil
}

// metaArguments are Terraform's own resource and data source arguments.
var metaArguments = map[string]bool{"count": true, "for_each": true, "provider": true, "depends_on": true, "lifecycle": true}

func unknownAttrs(path string, body *hclsyntax.Body, block *tfjson.SchemaBlock) []string {
	var out []string
	for _, name := range sortedAttrNames(body.Attributes) {
		if metaArguments[name] {
			continue
		}
		a, ok := block.Attributes[name]
		if !ok {
			if _, isBlock := block.NestedBlocks[name]; !isBlock {
				out = append(out, fmt.Sprintf("%s.%s is unknown", path, name))
			}
			continue
		}
		if a.AttributeNestedType != nil {
			out = append(out, unknownNested(path+"."+name, body.Attributes[name].Expr, a.AttributeNestedType.Attributes)...)
		}
	}
	for _, nb := range body.Blocks {
		if metaArguments[nb.Type] {
			continue
		}
		bt, ok := block.NestedBlocks[nb.Type]
		if !ok {
			out = append(out, fmt.Sprintf("%s.%s is unknown", path, nb.Type))
			continue
		}
		out = append(out, unknownAttrs(path+"."+nb.Type, nb.Body, bt.Block)...)
	}
	return out
}

// unknownNested checks the keys of literal objects (or lists of objects)
// assigned to a nested attribute.
func unknownNested(path string, expr hclsyntax.Expression, attrs map[string]*tfjson.SchemaAttribute) []string {
	var objs []*hclsyntax.ObjectConsExpr
	switch e := expr.(type) {
	case *hclsyntax.ObjectConsExpr:
		objs = append(objs, e)
	case *hclsyntax.TupleConsExpr:
		for _, item := range e.Exprs {
			if o, ok := item.(*hclsyntax.ObjectConsExpr); ok {
				objs = append(objs, o)
			}
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, o := range objs {
		for _, item := range o.Items {
			key := hcl.ExprAsKeyword(item.KeyExpr)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			a, ok := attrs[key]
			if !ok {
				out = append(out, fmt.Sprintf("%s.%s is unknown", path, key))
				continue
			}
			if a.AttributeNestedType != nil {
				out = append(out, unknownNested(path+"."+key, item.ValueExpr, a.AttributeNestedType.Attributes)...)
			}
		}
	}
	return out
}

func sortedAttrNames(attrs hclsyntax.Attributes) []string {
	names := make([]string, 0, len(attrs))
	for n := range attrs {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
