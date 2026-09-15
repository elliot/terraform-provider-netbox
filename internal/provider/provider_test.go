package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/elliot/terraform-provider-netbox/internal/client"
)

// ---------------------------------------------------------------------------
// test probe data source
//
// Terraform only configures providers that are referenced by at least one
// resource or data source, so the tests register a tiny data source that
// simply echoes ProviderData. It exists only in the test binary.
// ---------------------------------------------------------------------------

func init() {
	RegisterDataSource(func() datasource.DataSource { return &testProbeDataSource{} })
}

type testProbeDataSource struct {
	pd *ProviderData
}

type testProbeModel struct {
	ServerURL     types.String `tfsdk:"server_url"`
	NetBoxVersion types.String `tfsdk:"netbox_version"`
	UserAgent     types.String `tfsdk:"user_agent"`
	TokenPrefix   types.String `tfsdk:"token_prefix"`
}

func (d *testProbeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_test_probe"
}

func (d *testProbeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		Description: "Test-only probe exposing the configured provider data.",
		Attributes: map[string]dsschema.Attribute{
			"server_url":     dsschema.StringAttribute{Computed: true},
			"netbox_version": dsschema.StringAttribute{Computed: true},
			"user_agent":     dsschema.StringAttribute{Computed: true},
			"token_prefix":   dsschema.StringAttribute{Computed: true},
		},
	}
}

func (d *testProbeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	d.pd = pd
}

func (d *testProbeDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.pd == nil {
		resp.Diagnostics.AddError("Provider not configured", "ProviderData is nil")
		return
	}
	prefix, _, _ := strings.Cut(d.pd.Token, ".")
	m := testProbeModel{
		ServerURL:     types.StringValue(d.pd.ServerURL),
		NetBoxVersion: types.StringValue(d.pd.NetBoxVersion),
		UserAgent:     types.StringValue(d.pd.UserAgent),
		TokenPrefix:   types.StringValue(prefix),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

// ---------------------------------------------------------------------------
// fake NetBox
// ---------------------------------------------------------------------------

const testToken = "nbt_x.y"

// newFakeNetBox serves /api/status/ and enforces the Bearer token.
func newFakeNetBox(t *testing.T, version string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer "+testToken {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"detail":"Invalid token."}`))
			return
		}
		if r.URL.Path != "/api/status/" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"detail":"Not found."}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"django-version": "5.2.1",
			"netbox-version": version,
			"plugins":        map[string]any{},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// clearNetBoxEnv makes sure a developer's shell environment does not leak
// into the tests.
func clearNetBoxEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{EnvServerURL, EnvAPIToken, EnvRequestsPerSecond, EnvRequestBurst,
		EnvRequestDelayMs, EnvWriteDelayMs, EnvSerializeRequests, EnvMaxRetries, EnvRetryWaitMinMs,
		EnvRetryWaitMaxMs, EnvRequestTimeoutMs, EnvAllowInsecureHTTPS, EnvSkipVersionCheck, EnvUserAgent} {
		t.Setenv(k, "")
	}
}

const probeConfig = `
data "netbox_test_probe" "t" {}
`

// ---------------------------------------------------------------------------
// pure unit tests (no Terraform CLI)
// ---------------------------------------------------------------------------

func TestResolveSettings(t *testing.T) {
	ctx := context.Background()
	env := func(vals map[string]string) func(string) string {
		return func(k string) string { return vals[k] }
	}

	t.Run("defaults", func(t *testing.T) {
		m := providerModel{
			ServerURL: types.StringValue("https://nb.example.com/api/"),
			APIToken:  types.StringValue(testToken),
			Headers:   types.MapNull(types.StringType),
		}
		s, diags := resolveSettings(ctx, m, "1.2.3", env(nil))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		c := s.client
		if c.ServerURL != "https://nb.example.com" {
			t.Errorf("ServerURL = %q, want /api and trailing slash stripped", c.ServerURL)
		}
		if c.MaxRetries != 4 || c.RetryWaitMin != time.Second || c.RetryWaitMax != 30*time.Second ||
			c.RequestTimeout != 60*time.Second || c.Burst != 10 || c.RequestsPerSecond != 0 {
			t.Errorf("defaults not applied: %+v", c)
		}
		if c.UserAgent != "terraform-provider-netbox/1.2.3" {
			t.Errorf("UserAgent = %q", c.UserAgent)
		}
		if s.skipVersionCheck {
			t.Error("skipVersionCheck should default to false")
		}
	})

	t.Run("env fallback", func(t *testing.T) {
		m := providerModel{Headers: types.MapNull(types.StringType)}
		s, diags := resolveSettings(ctx, m, "dev", env(map[string]string{
			EnvServerURL:          "https://env.example.com/",
			EnvAPIToken:           "nbt_env.secret",
			EnvRequestsPerSecond:  "2.5",
			EnvRequestBurst:       "3",
			EnvRequestDelayMs:     "150",
			EnvWriteDelayMs:       "250",
			EnvSerializeRequests:  "true",
			EnvMaxRetries:         "7",
			EnvRetryWaitMinMs:     "10",
			EnvRetryWaitMaxMs:     "20",
			EnvRequestTimeoutMs:   "5000",
			EnvAllowInsecureHTTPS: "1",
			EnvSkipVersionCheck:   "yes",
			EnvUserAgent:          "custom/1",
		}))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		c := s.client
		checks := map[string]bool{
			"server_url":  c.ServerURL == "https://env.example.com",
			"token":       c.Token == "nbt_env.secret",
			"rps":         c.RequestsPerSecond == 2.5,
			"burst":       c.Burst == 3,
			"delay":       c.RequestDelay == 150*time.Millisecond,
			"write delay": c.WriteDelay == 250*time.Millisecond,
			"serialize":   c.SerializeRequests,
			"retries":     c.MaxRetries == 7,
			"wait min":    c.RetryWaitMin == 10*time.Millisecond,
			"wait max":    c.RetryWaitMax == 20*time.Millisecond,
			"timeout":     c.RequestTimeout == 5*time.Second,
			"insecure":    c.InsecureSkipVerify,
			"skip check":  s.skipVersionCheck,
			"user agent":  c.UserAgent == "custom/1",
		}
		for name, ok := range checks {
			if !ok {
				t.Errorf("%s not resolved from env: %+v", name, c)
			}
		}
	})

	t.Run("config wins over env", func(t *testing.T) {
		m := providerModel{
			ServerURL:  types.StringValue("https://cfg.example.com"),
			APIToken:   types.StringValue("nbt_cfg.secret"),
			MaxRetries: types.Int64Value(0),
			Headers: types.MapValueMust(types.StringType, map[string]attr.Value{
				"X-Proxy": types.StringValue("yes"),
			}),
		}
		s, diags := resolveSettings(ctx, m, "dev", env(map[string]string{
			EnvServerURL:  "https://env.example.com",
			EnvAPIToken:   "nbt_env.secret",
			EnvMaxRetries: "9",
		}))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if s.client.ServerURL != "https://cfg.example.com" || s.client.Token != "nbt_cfg.secret" || s.client.MaxRetries != 0 {
			t.Errorf("config did not take precedence: %+v", s.client)
		}
		if s.client.Headers["X-Proxy"] != "yes" {
			t.Errorf("headers not resolved: %v", s.client.Headers)
		}
	})

	t.Run("missing server_url", func(t *testing.T) {
		m := providerModel{APIToken: types.StringValue(testToken), Headers: types.MapNull(types.StringType)}
		_, diags := resolveSettings(ctx, m, "dev", env(nil))
		assertDiagContains(t, diags, "Missing NetBox server URL")
	})

	t.Run("missing token", func(t *testing.T) {
		m := providerModel{ServerURL: types.StringValue("https://x"), Headers: types.MapNull(types.StringType)}
		_, diags := resolveSettings(ctx, m, "dev", env(nil))
		assertDiagContains(t, diags, "Missing NetBox API token")
	})

	t.Run("v1 token rejected", func(t *testing.T) {
		for _, tok := range []string{"Token abc", "0123456789abcdef0123456789abcdef01234567", "nbt_nodot", "nbt_.x", "nbt_x."} {
			m := providerModel{
				ServerURL: types.StringValue("https://x"),
				APIToken:  types.StringValue(tok),
				Headers:   types.MapNull(types.StringType),
			}
			_, diags := resolveSettings(ctx, m, "dev", env(nil))
			assertDiagContains(t, diags, "Unsupported NetBox API token format")
		}
	})

	t.Run("bad env values", func(t *testing.T) {
		m := providerModel{
			ServerURL: types.StringValue("https://x"),
			APIToken:  types.StringValue(testToken),
			Headers:   types.MapNull(types.StringType),
		}
		for envName, val := range map[string]string{
			EnvMaxRetries:        "many",
			EnvRequestsPerSecond: "fast",
			EnvSerializeRequests: "maybe",
		} {
			_, diags := resolveSettings(ctx, m, "dev", env(map[string]string{envName: val}))
			assertDiagContains(t, diags, "Invalid value in environment variable "+envName)
		}
	})

	t.Run("unknown value", func(t *testing.T) {
		m := providerModel{
			ServerURL: types.StringUnknown(),
			APIToken:  types.StringValue(testToken),
			Headers:   types.MapNull(types.StringType),
		}
		_, diags := resolveSettings(ctx, m, "dev", env(nil))
		assertDiagContains(t, diags, "Unknown provider configuration value")
	})

	t.Run("validation errors surface", func(t *testing.T) {
		m := providerModel{
			ServerURL:      types.StringValue("https://x"),
			APIToken:       types.StringValue(testToken),
			RetryWaitMinMs: types.Int64Value(5000),
			RetryWaitMaxMs: types.Int64Value(100),
			Headers:        types.MapNull(types.StringType),
		}
		_, diags := resolveSettings(ctx, m, "dev", env(nil))
		assertDiagContains(t, diags, "Invalid NetBox provider configuration")
	})
}

func assertDiagContains(t *testing.T, diags diag.Diagnostics, summary string) {
	t.Helper()
	if !diags.HasError() {
		t.Fatalf("expected an error diagnostic containing %q", summary)
	}
	for _, d := range diags.Errors() {
		if strings.Contains(d.Summary(), summary) {
			return
		}
	}
	t.Fatalf("no error diagnostic with summary %q in %v", summary, diags)
}

func TestIsSupportedVersion(t *testing.T) {
	cases := map[string]bool{
		"4.7.0": true, "4.7.12": true, "v4.7.1": true, "4.7.0-Docker-3.5.0": true, "4.7": true,
		"4.6.9": false, "4.70.1": false, "5.0.0": false, "": false,
	}
	for v, want := range cases {
		if got := isSupportedVersion(v); got != want {
			t.Errorf("isSupportedVersion(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestCheckServer(t *testing.T) {
	ctx := context.Background()
	newClient := func(t *testing.T, srv *httptest.Server, token string) *http.Client {
		t.Helper()
		cfg := client.Defaults()
		cfg.ServerURL = srv.URL
		cfg.Token = token
		cfg.RetryWaitMin, cfg.RetryWaitMax = time.Millisecond, 2*time.Millisecond
		hc, err := client.NewHTTPClient(cfg)
		if err != nil {
			t.Fatal(err)
		}
		return hc
	}

	t.Run("ok", func(t *testing.T) {
		srv := newFakeNetBox(t, "4.7.3")
		var diags diag.Diagnostics
		v := checkServer(ctx, newClient(t, srv, testToken), srv.URL, &diags)
		if v != "4.7.3" || len(diags) != 0 {
			t.Fatalf("version = %q diags = %v", v, diags)
		}
	})
	t.Run("old version warns", func(t *testing.T) {
		srv := newFakeNetBox(t, "4.6.0")
		var diags diag.Diagnostics
		v := checkServer(ctx, newClient(t, srv, testToken), srv.URL, &diags)
		if v != "4.6.0" || diags.HasError() || diags.WarningsCount() != 1 {
			t.Fatalf("version = %q diags = %v", v, diags)
		}
	})
	t.Run("bad token errors", func(t *testing.T) {
		srv := newFakeNetBox(t, "4.7.0")
		var diags diag.Diagnostics
		checkServer(ctx, newClient(t, srv, "nbt_wrong.token"), srv.URL, &diags)
		assertDiagContains(t, diags, "NetBox rejected the API token")
	})
	t.Run("unreachable warns", func(t *testing.T) {
		srv := newFakeNetBox(t, "4.7.0")
		hc := newClient(t, srv, testToken)
		srv.Close()
		var diags diag.Diagnostics
		checkServer(ctx, hc, srv.URL, &diags)
		if diags.HasError() || diags.WarningsCount() != 1 {
			t.Fatalf("diags = %v", diags)
		}
	})
	t.Run("not netbox warns", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html>hello</html>"))
		}))
		defer srv.Close()
		var diags diag.Diagnostics
		checkServer(ctx, newClient(t, srv, testToken), srv.URL, &diags)
		if diags.HasError() || diags.WarningsCount() != 1 {
			t.Fatalf("diags = %v", diags)
		}
	})
}

func TestRegistryReturnsCopies(t *testing.T) {
	ds := registeredDataSources()
	if len(ds) == 0 {
		t.Fatal("test probe data source should be registered")
	}
	ds[0] = nil
	if registeredDataSources()[0] == nil {
		t.Fatal("registeredDataSources must return a copy")
	}
	p := New("t")()
	if len(p.DataSources(context.Background())) != len(registeredDataSources()) {
		t.Fatal("provider must expose registered data sources")
	}
	if len(p.Resources(context.Background())) != len(registeredResources()) {
		t.Fatal("provider must expose registered resources")
	}
}

// ---------------------------------------------------------------------------
// Terraform-driven tests (terraform-plugin-testing, no TF_ACC needed)
// ---------------------------------------------------------------------------

func TestUnitProviderConfigure_Explicit(t *testing.T) {
	clearNetBoxEnv(t)
	srv := newFakeNetBox(t, "4.7.0")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
provider "netbox" {
  server_url = %q
  api_token  = %q
  user_agent = "unit-test/1"
}
`, srv.URL+"/api/", testToken) + probeConfig,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.netbox_test_probe.t", "server_url", srv.URL),
				resource.TestCheckResourceAttr("data.netbox_test_probe.t", "netbox_version", "4.7.0"),
				resource.TestCheckResourceAttr("data.netbox_test_probe.t", "user_agent", "unit-test/1"),
				resource.TestCheckResourceAttr("data.netbox_test_probe.t", "token_prefix", "nbt_x"),
			),
		}},
	})
}

func TestUnitProviderConfigure_FromEnv(t *testing.T) {
	clearNetBoxEnv(t)
	srv := newFakeNetBox(t, "4.7.1")
	t.Setenv(EnvServerURL, srv.URL)
	t.Setenv(EnvAPIToken, testToken)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `provider "netbox" {}` + probeConfig,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.netbox_test_probe.t", "server_url", srv.URL),
				resource.TestCheckResourceAttr("data.netbox_test_probe.t", "netbox_version", "4.7.1"),
			),
		}},
	})
}

func TestUnitProviderConfigure_SkipVersionCheck(t *testing.T) {
	clearNetBoxEnv(t)
	t.Setenv(EnvServerURL, "http://127.0.0.1:9") // nothing listens here
	t.Setenv(EnvAPIToken, testToken)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `provider "netbox" { skip_version_check = true }` + probeConfig,
			Check:  resource.TestCheckResourceAttr("data.netbox_test_probe.t", "netbox_version", ""),
		}},
	})
}

func TestUnitProviderConfigure_V1TokenRejected(t *testing.T) {
	clearNetBoxEnv(t)
	srv := newFakeNetBox(t, "4.7.0")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
provider "netbox" {
  server_url = %q
  api_token  = "Token abc"
}
`, srv.URL) + probeConfig,
			ExpectError: regexp.MustCompile(`Unsupported NetBox API token format`),
		}},
	})
}

func TestUnitProviderConfigure_MissingServerURL(t *testing.T) {
	clearNetBoxEnv(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
provider "netbox" {
  api_token = %q
}
`, testToken) + probeConfig,
			ExpectError: regexp.MustCompile(`Missing NetBox server URL`),
		}},
	})
}

func TestUnitProviderConfigure_RejectedToken(t *testing.T) {
	clearNetBoxEnv(t)
	srv := newFakeNetBox(t, "4.7.0")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
provider "netbox" {
  server_url        = %q
  api_token         = "nbt_wrong.secret"
  retry_wait_min_ms = 1
  retry_wait_max_ms = 2
}
`, srv.URL) + probeConfig,
			ExpectError: regexp.MustCompile(`NetBox rejected the API token`),
		}},
	})
}
