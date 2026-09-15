// Package provider implements the Terraform provider for NetBox 4.7.
//
// Resources and data sources are contributed by generated packages through
// RegisterResource / RegisterDataSource; this package owns the provider
// schema, configuration resolution (config > environment > default), the
// HTTP client construction and the one-time /api/status/ version check.
package provider

import (
	"github.com/elliot/terraform-provider-netbox/netbox"

	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/elliot/terraform-provider-netbox/internal/client"
)

// Environment variables understood by the provider.
const (
	EnvServerURL          = "NETBOX_SERVER_URL"
	EnvAPIToken           = "NETBOX_API_TOKEN" //nolint:gosec // environment variable name, not a credential
	EnvRequestsPerSecond  = "NETBOX_REQUESTS_PER_SECOND"
	EnvRequestBurst       = "NETBOX_REQUEST_BURST"
	EnvRequestDelayMs     = "NETBOX_REQUEST_DELAY_MS"
	EnvWriteDelayMs       = "NETBOX_WRITE_DELAY_MS"
	EnvSerializeRequests  = "NETBOX_SERIALIZE_REQUESTS"
	EnvMaxRetries         = "NETBOX_MAX_RETRIES"
	EnvRetryWaitMinMs     = "NETBOX_RETRY_WAIT_MIN_MS"
	EnvRetryWaitMaxMs     = "NETBOX_RETRY_WAIT_MAX_MS"
	EnvRequestTimeoutMs   = "NETBOX_REQUEST_TIMEOUT_MS"
	EnvAllowInsecureHTTPS = "NETBOX_ALLOW_INSECURE_HTTPS"
	EnvSkipVersionCheck   = "NETBOX_SKIP_VERSION_CHECK"
	EnvUserAgent          = "NETBOX_USER_AGENT"
)

// SupportedNetBoxMajorMinor is the NetBox release line this provider targets.
const SupportedNetBoxMajorMinor = "4.7"

// versionCheckTimeout bounds the /api/status/ call made during Configure.
const versionCheckTimeout = 30 * time.Second

// Ensure the provider satisfies the framework interfaces.
var _ provider.Provider = (*netboxProvider)(nil)

// netboxProvider is the provider implementation.
type netboxProvider struct {
	version string
}

// New returns a provider factory for the given build version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &netboxProvider{version: version}
	}
}

// providerModel maps the provider schema to Go values.
type providerModel struct {
	ServerURL          types.String  `tfsdk:"server_url"`
	APIToken           types.String  `tfsdk:"api_token"`
	RequestsPerSecond  types.Float64 `tfsdk:"requests_per_second"`
	RequestBurst       types.Int64   `tfsdk:"request_burst"`
	RequestDelayMs     types.Int64   `tfsdk:"request_delay_ms"`
	WriteDelayMs       types.Int64   `tfsdk:"write_delay_ms"`
	SerializeRequests  types.Bool    `tfsdk:"serialize_requests"`
	MaxRetries         types.Int64   `tfsdk:"max_retries"`
	RetryWaitMinMs     types.Int64   `tfsdk:"retry_wait_min_ms"`
	RetryWaitMaxMs     types.Int64   `tfsdk:"retry_wait_max_ms"`
	RequestTimeoutMs   types.Int64   `tfsdk:"request_timeout_ms"`
	AllowInsecureHTTPS types.Bool    `tfsdk:"allow_insecure_https"`
	Headers            types.Map     `tfsdk:"headers"`
	SkipVersionCheck   types.Bool    `tfsdk:"skip_version_check"`
	UserAgent          types.String  `tfsdk:"user_agent"`
}

// Metadata implements provider.Provider.
func (p *netboxProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "netbox"
	resp.Version = p.version
}

// Schema implements provider.Provider.
func (p *netboxProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The NetBox provider manages objects in a [NetBox](https://netbox.dev) " +
			SupportedNetBoxMajorMinor + " instance through its REST API. Every attribute can also be " +
			"supplied through the environment variable named in its description; explicit configuration " +
			"takes precedence over the environment.",
		Attributes: map[string]schema.Attribute{
			"server_url": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Base URL of the NetBox instance, e.g. `https://netbox.example.com`. " +
					"A trailing `/` or `/api` is ignored. Env: `" + EnvServerURL + "`.",
			},
			"api_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				MarkdownDescription: "NetBox **v2** API token in the form `nbt_<key>.<secret>`, sent as " +
					"`Authorization: Bearer ...`. Legacy v1 tokens are not supported. Env: `" + EnvAPIToken + "`.",
			},
			"requests_per_second": schema.Float64Attribute{
				Optional:   true,
				Validators: []validator.Float64{float64validator.AtLeast(0)},
				MarkdownDescription: "Sustained request rate of the client-side token-bucket limiter shared by all " +
					"resources. `0` (the default) disables rate limiting. Env: `" + EnvRequestsPerSecond + "`.",
			},
			"request_burst": schema.Int64Attribute{
				Optional:   true,
				Validators: []validator.Int64{int64validator.AtLeast(1)},
				MarkdownDescription: "Burst capacity of the rate limiter. Only used when `requests_per_second` > 0. " +
					"Default `" + strconv.Itoa(client.DefaultBurst) + "`. Env: `" + EnvRequestBurst + "`.",
			},
			"request_delay_ms": schema.Int64Attribute{
				Optional:   true,
				Validators: []validator.Int64{int64validator.AtLeast(0)},
				MarkdownDescription: "Fixed artificial delay, in milliseconds, applied before every request. " +
					"Default `0`. Env: `" + EnvRequestDelayMs + "`.",
			},
			"write_delay_ms": schema.Int64Attribute{
				Optional:   true,
				Validators: []validator.Int64{int64validator.AtLeast(0)},
				MarkdownDescription: "Additional delay, in milliseconds, applied before every POST, PUT, PATCH " +
					"and DELETE request. Default `0`. Env: `" + EnvWriteDelayMs + "`.",
			},
			"serialize_requests": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "When `true`, at most one request is in flight at any time. " +
					"Default `false`. Env: `" + EnvSerializeRequests + "`.",
			},
			"max_retries": schema.Int64Attribute{
				Optional:   true,
				Validators: []validator.Int64{int64validator.AtLeast(0)},
				MarkdownDescription: "Number of retries after the first attempt for network errors, `429` and `5xx` " +
					"responses (except `501`). Default `" + strconv.Itoa(client.DefaultMaxRetries) + "`. Env: `" + EnvMaxRetries + "`.",
			},
			"retry_wait_min_ms": schema.Int64Attribute{
				Optional:   true,
				Validators: []validator.Int64{int64validator.AtLeast(0)},
				MarkdownDescription: "Minimum backoff between retries, in milliseconds. Backoff is exponential with " +
					"full jitter; `Retry-After` is honoured on `429`/`503`. Default `" +
					strconv.FormatInt(client.DefaultRetryWaitMin.Milliseconds(), 10) + "`. Env: `" + EnvRetryWaitMinMs + "`.",
			},
			"retry_wait_max_ms": schema.Int64Attribute{
				Optional:   true,
				Validators: []validator.Int64{int64validator.AtLeast(0)},
				MarkdownDescription: "Maximum backoff between retries, in milliseconds (also caps `Retry-After`). " +
					"Default `" + strconv.FormatInt(client.DefaultRetryWaitMax.Milliseconds(), 10) + "`. Env: `" + EnvRetryWaitMaxMs + "`.",
			},
			"request_timeout_ms": schema.Int64Attribute{
				Optional:   true,
				Validators: []validator.Int64{int64validator.AtLeast(0)},
				MarkdownDescription: "Timeout for a single request attempt, in milliseconds, excluding time spent " +
					"waiting on the rate limiter. `0` disables the timeout. Default `" +
					strconv.FormatInt(client.DefaultRequestTimeout.Milliseconds(), 10) + "`. Env: `" + EnvRequestTimeoutMs + "`.",
			},
			"allow_insecure_https": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Skip TLS certificate verification. Default `false`. Env: `" + EnvAllowInsecureHTTPS + "`.",
			},
			"headers": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				MarkdownDescription: "Extra HTTP headers sent with every request (for example for a reverse proxy). " +
					"`Authorization` cannot be overridden.",
			},
			"skip_version_check": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "Skip the `GET /api/status/` call made when the provider is configured. " +
					"By default the provider verifies the token and warns when the server is not NetBox " +
					SupportedNetBoxMajorMinor + ".x. Default `false`. Env: `" + EnvSkipVersionCheck + "`.",
			},
			"user_agent": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Value of the `User-Agent` header. Default `terraform-provider-netbox/<version>`. " +
					"Env: `" + EnvUserAgent + "`.",
			},
		},
	}
}

// settings is the fully resolved provider configuration.
type settings struct {
	client           client.Config
	skipVersionCheck bool
}

// Configure implements provider.Provider.
func (p *netboxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var model providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	s, diags := resolveSettings(ctx, model, p.version, os.Getenv)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpClient, err := client.NewHTTPClient(s.client)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create NetBox HTTP client", err.Error())
		return
	}

	pd := &ProviderData{
		HTTPClient: httpClient,
		ServerURL:  s.client.ServerURL,
		Token:      s.client.Token,
		UserAgent:  s.client.UserAgent,
		API:        netbox.NewClient(s.client.ServerURL, httpClient, ""),
	}
	pd.API.GetConfig().UserAgent = s.client.UserAgent

	if s.skipVersionCheck {
		tflog.Debug(ctx, "netbox: skipping version check")
	} else {
		pd.NetBoxVersion = checkServer(ctx, httpClient, s.client.ServerURL, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Info(ctx, "netbox: provider configured", map[string]any{
		"server_url":          pd.ServerURL,
		"netbox_version":      pd.NetBoxVersion,
		"requests_per_second": s.client.RequestsPerSecond,
		"request_delay_ms":    s.client.RequestDelay.Milliseconds(),
		"write_delay_ms":      s.client.WriteDelay.Milliseconds(),
		"serialize_requests":  s.client.SerializeRequests,
		"max_retries":         s.client.MaxRetries,
	})

	resp.ResourceData = pd
	resp.DataSourceData = pd
}

// Resources implements provider.Provider.
func (p *netboxProvider) Resources(_ context.Context) []func() resource.Resource {
	return registeredResources()
}

// DataSources implements provider.Provider.
func (p *netboxProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return registeredDataSources()
}

// resolveSettings merges explicit configuration, environment variables and
// defaults, validating as it goes. getenv is injectable for tests.
func resolveSettings(ctx context.Context, m providerModel, version string, getenv func(string) string) (settings, diag.Diagnostics) {
	var diags diag.Diagnostics
	s := settings{client: client.Defaults()}

	// Unknown values (e.g. references to resources not yet created) cannot be
	// used to configure the provider.
	for _, a := range []struct {
		name string
		v    attr.Value
	}{
		{"server_url", m.ServerURL}, {"api_token", m.APIToken},
		{"requests_per_second", m.RequestsPerSecond}, {"request_burst", m.RequestBurst},
		{"request_delay_ms", m.RequestDelayMs}, {"write_delay_ms", m.WriteDelayMs},
		{"serialize_requests", m.SerializeRequests}, {"max_retries", m.MaxRetries},
		{"retry_wait_min_ms", m.RetryWaitMinMs}, {"retry_wait_max_ms", m.RetryWaitMaxMs},
		{"request_timeout_ms", m.RequestTimeoutMs}, {"allow_insecure_https", m.AllowInsecureHTTPS},
		{"headers", m.Headers}, {"skip_version_check", m.SkipVersionCheck}, {"user_agent", m.UserAgent},
	} {
		if a.v.IsUnknown() {
			diags.AddAttributeError(path.Root(a.name),
				"Unknown provider configuration value",
				fmt.Sprintf("The provider cannot be configured because %q is not yet known. "+
					"Set it to a static value or supply it through %s.", a.name, envFor(a.name)))
		}
	}
	if diags.HasError() {
		return s, diags
	}

	r := resolver{getenv: getenv, diags: &diags}

	s.client.ServerURL = client.NormalizeServerURL(r.str(m.ServerURL, EnvServerURL))
	if s.client.ServerURL == "" {
		diags.AddAttributeError(path.Root("server_url"),
			"Missing NetBox server URL",
			"Set server_url in the provider configuration or the "+EnvServerURL+" environment variable.")
	}

	s.client.Token = strings.TrimSpace(r.str(m.APIToken, EnvAPIToken))
	switch {
	case s.client.Token == "":
		diags.AddAttributeError(path.Root("api_token"),
			"Missing NetBox API token",
			"Set api_token in the provider configuration or the "+EnvAPIToken+" environment variable.")
	case !isV2Token(s.client.Token):
		diags.AddAttributeError(path.Root("api_token"),
			"Unsupported NetBox API token format",
			"This provider supports only NetBox v2 API tokens, which look like nbt_<key>.<secret> and are sent "+
				"as \"Authorization: Bearer ...\". Legacy v1 tokens (\"Authorization: Token ...\") are not supported. "+
				"Create a v2 token in the NetBox UI (Admin > API Tokens) or with POST /api/users/tokens/provision/ "+
				"using \"version\": 2; the secret is shown only once.")
	}

	s.client.RequestsPerSecond = r.f64(m.RequestsPerSecond, EnvRequestsPerSecond, s.client.RequestsPerSecond)
	s.client.Burst = int(r.i64(m.RequestBurst, EnvRequestBurst, int64(s.client.Burst)))
	s.client.RequestDelay = time.Duration(r.i64(m.RequestDelayMs, EnvRequestDelayMs, 0)) * time.Millisecond
	s.client.WriteDelay = time.Duration(r.i64(m.WriteDelayMs, EnvWriteDelayMs, 0)) * time.Millisecond
	s.client.SerializeRequests = r.boolean(m.SerializeRequests, EnvSerializeRequests, false)
	s.client.MaxRetries = int(r.i64(m.MaxRetries, EnvMaxRetries, int64(s.client.MaxRetries)))
	s.client.RetryWaitMin = time.Duration(r.i64(m.RetryWaitMinMs, EnvRetryWaitMinMs, s.client.RetryWaitMin.Milliseconds())) * time.Millisecond
	s.client.RetryWaitMax = time.Duration(r.i64(m.RetryWaitMaxMs, EnvRetryWaitMaxMs, s.client.RetryWaitMax.Milliseconds())) * time.Millisecond
	s.client.RequestTimeout = time.Duration(r.i64(m.RequestTimeoutMs, EnvRequestTimeoutMs, s.client.RequestTimeout.Milliseconds())) * time.Millisecond
	s.client.InsecureSkipVerify = r.boolean(m.AllowInsecureHTTPS, EnvAllowInsecureHTTPS, false)
	s.skipVersionCheck = r.boolean(m.SkipVersionCheck, EnvSkipVersionCheck, false)

	s.client.UserAgent = r.str(m.UserAgent, EnvUserAgent)
	if s.client.UserAgent == "" {
		s.client.UserAgent = "terraform-provider-netbox/" + version
	}

	if !m.Headers.IsNull() {
		headers := make(map[string]string, len(m.Headers.Elements()))
		diags.Append(m.Headers.ElementsAs(ctx, &headers, false)...)
		s.client.Headers = headers
	}

	if diags.HasError() {
		return s, diags
	}
	if err := s.client.Validate(); err != nil {
		diags.AddError("Invalid NetBox provider configuration", err.Error())
	}
	return s, diags
}

// isV2Token reports whether tok looks like a NetBox v2 token (nbt_<key>.<secret>).
func isV2Token(tok string) bool {
	if !strings.HasPrefix(tok, "nbt_") {
		return false
	}
	rest := strings.TrimPrefix(tok, "nbt_")
	key, secret, ok := strings.Cut(rest, ".")
	return ok && key != "" && secret != ""
}

// envFor maps an attribute name to its environment variable.
func envFor(attrName string) string {
	return "NETBOX_" + strings.ToUpper(attrName)
}

// resolver picks config > env > default for scalar attributes, recording
// parse errors against the attribute path.
type resolver struct {
	getenv func(string) string
	diags  *diag.Diagnostics
}

func (r resolver) envError(env string, err error) {
	r.diags.AddAttributeError(path.Root(strings.ToLower(strings.TrimPrefix(env, "NETBOX_"))),
		"Invalid value in environment variable "+env, err.Error())
}

func (r resolver) str(v types.String, env string) string {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueString()
	}
	return r.getenv(env)
}

func (r resolver) i64(v types.Int64, env string, def int64) int64 {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueInt64()
	}
	if raw := strings.TrimSpace(r.getenv(env)); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			r.envError(env, fmt.Errorf("expected an integer, got %q", raw))
			return def
		}
		return n
	}
	return def
}

func (r resolver) f64(v types.Float64, env string, def float64) float64 {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueFloat64()
	}
	if raw := strings.TrimSpace(r.getenv(env)); raw != "" {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			r.envError(env, fmt.Errorf("expected a number, got %q", raw))
			return def
		}
		return f
	}
	return def
}

func (r resolver) boolean(v types.Bool, env string, def bool) bool {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueBool()
	}
	if raw := strings.TrimSpace(r.getenv(env)); raw != "" {
		b, ok := parseBool(raw)
		if !ok {
			r.envError(env, fmt.Errorf("expected true or false, got %q", raw))
			return def
		}
		return b
	}
	return def
}

// parseBool accepts everything strconv.ParseBool does plus yes/no/on/off.
func parseBool(raw string) (bool, bool) {
	if b, err := strconv.ParseBool(raw); err == nil {
		return b, true
	}
	switch strings.ToLower(raw) {
	case "yes", "on", "y":
		return true, true
	case "no", "off", "n":
		return false, true
	}
	return false, false
}

// statusResponse is the subset of GET /api/status/ the provider looks at.
type statusResponse struct {
	NetBoxVersion string `json:"netbox-version"`
}

// checkServer calls GET /api/status/ once. A 401/403 is reported as an error
// (bad token); every other failure is a warning; an unexpected NetBox release
// line is a warning. It returns the reported version, if any.
func checkServer(ctx context.Context, hc *http.Client, serverURL string, diags *diag.Diagnostics) string {
	ctx, cancel := context.WithTimeout(ctx, versionCheckTimeout)
	defer cancel()

	statusURL := serverURL + "/api/status/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		diags.AddWarning("Could not verify NetBox version", fmt.Sprintf("Building request for %s failed: %s", statusURL, err))
		return ""
	}

	resp, err := hc.Do(req)
	if err != nil {
		diags.AddWarning("Could not verify NetBox version",
			fmt.Sprintf("GET %s failed: %s. The provider will continue, but subsequent API calls may fail. "+
				"Set skip_version_check = true to silence this warning.", statusURL, err))
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		apiErr := client.ParseAPIError(resp, body)
		diags.AddAttributeError(path.Root("api_token"),
			"NetBox rejected the API token",
			fmt.Sprintf("GET %s returned %d %s: %s. Check that api_token (or %s) is a valid, enabled v2 token "+
				"for this server and that its IP allow-list (if any) includes this host.",
				statusURL, resp.StatusCode, http.StatusText(resp.StatusCode), apiErr.Detail, EnvAPIToken))
		return ""
	case resp.StatusCode != http.StatusOK:
		apiErr := client.ParseAPIError(resp, body)
		diags.AddWarning("Could not verify NetBox version",
			fmt.Sprintf("GET %s returned %d %s: %s. Set skip_version_check = true to silence this warning.",
				statusURL, resp.StatusCode, http.StatusText(resp.StatusCode), apiErr.Detail))
		return ""
	}

	var st statusResponse
	if err := json.Unmarshal(body, &st); err != nil {
		diags.AddWarning("Could not verify NetBox version",
			fmt.Sprintf("GET %s returned an unparseable body: %s. Is server_url pointing at a NetBox instance?", statusURL, err))
		return ""
	}
	if st.NetBoxVersion == "" {
		diags.AddWarning("Could not verify NetBox version",
			fmt.Sprintf("GET %s did not report a netbox-version. Is server_url pointing at a NetBox instance?", statusURL))
		return ""
	}

	if !isSupportedVersion(st.NetBoxVersion) {
		diags.AddWarning("Unsupported NetBox version",
			fmt.Sprintf("The server reports NetBox %s but this provider targets NetBox %s.x. "+
				"Some resources or attributes may not work as expected. Set skip_version_check = true to silence this warning.",
				st.NetBoxVersion, SupportedNetBoxMajorMinor))
	}
	tflog.Debug(ctx, "netbox: server version verified", map[string]any{"netbox_version": st.NetBoxVersion})
	return st.NetBoxVersion
}

// isSupportedVersion reports whether v belongs to the supported release line.
// NetBox reports versions such as "4.7.0", "4.7.2-Docker-3.5.0" or "v4.7.0".
func isSupportedVersion(v string) bool {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	return v == SupportedNetBoxMajorMinor || strings.HasPrefix(v, SupportedNetBoxMajorMinor+".") ||
		strings.HasPrefix(v, SupportedNetBoxMajorMinor+"-")
}
