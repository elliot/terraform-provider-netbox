package upgrade

import (
	"context"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func noop(context.Context, resource.UpgradeStateRequest, *resource.UpgradeStateResponse) {}

func TestRegistryForAndValidate(t *testing.T) {
	r := &Registry{}
	if got := r.For("netbox_x", 2); len(got) != 0 {
		t.Fatalf("empty registry returned %v", got)
	}
	if err := r.Validate("netbox_x", 0); err != nil {
		t.Fatalf("version 0 needs no upgraders: %v", err)
	}
	if err := r.Validate("netbox_x", 2); err == nil || !strings.Contains(err.Error(), "[0 1]") {
		t.Fatalf("missing upgraders not reported: %v", err)
	}
	r.Register("netbox_x", 0, resource.StateUpgrader{StateUpgrader: noop})
	r.Register("netbox_x", 1, resource.StateUpgrader{StateUpgrader: noop})
	if err := r.Validate("netbox_x", 2); err != nil {
		t.Fatalf("complete upgraders: %v", err)
	}
	if got := r.For("netbox_x", 2); len(got) != 2 {
		t.Fatalf("For returned %d upgraders, want 2", len(got))
	}
	// After a rollback to version 1, the upgrader from 1 is stale.
	if got := r.For("netbox_x", 1); len(got) != 1 {
		t.Fatalf("For must drop upgraders at or above the current version, got %v", got)
	}
	if err := r.Validate("netbox_x", 1); err == nil || !strings.Contains(err.Error(), "[1]") {
		t.Fatalf("stale upgrader not reported: %v", err)
	}
	if got := r.Types(); len(got) != 1 || got[0] != "netbox_x" {
		t.Fatalf("Types() = %v", got)
	}
	// For returns a copy.
	delete(r.For("netbox_x", 2), 0)
	if len(r.For("netbox_x", 2)) != 2 {
		t.Fatal("For must return a copy")
	}
}

func TestRegistryRegisterPanics(t *testing.T) {
	for name, f := range map[string]func(r *Registry){
		"duplicate": func(r *Registry) {
			r.Register("netbox_x", 0, resource.StateUpgrader{StateUpgrader: noop})
			r.Register("netbox_x", 0, resource.StateUpgrader{StateUpgrader: noop})
		},
		"negative version": func(r *Registry) { r.Register("netbox_x", -1, resource.StateUpgrader{StateUpgrader: noop}) },
		"no function":      func(r *Registry) { r.Register("netbox_x", 0, resource.StateUpgrader{}) },
		"no type":          func(r *Registry) { r.Register("", 0, resource.StateUpgrader{StateUpgrader: noop}) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected a panic")
				}
			}()
			f(&Registry{})
		})
	}
}

// fakeResource mirrors what the generator emits for `schema_version: 1`: a
// schema Version and an UpgradeState that delegates to a registry.
//
// Version 0 stored `port` as a string; version 1 stores it as a number and
// renames `label` to `name`.
type fakeResource struct{ reg *Registry }

var _ resource.ResourceWithUpgradeState = fakeResource{}

func (fakeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fake"
}

func (fakeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Attributes: map[string]schema.Attribute{
			"id":    schema.Int64Attribute{Computed: true},
			"name":  schema.StringAttribute{Optional: true},
			"label": schema.StringAttribute{Optional: true, DeprecationMessage: "Use name."},
			"port":  schema.Int64Attribute{Optional: true},
		},
	}
}

func (f fakeResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return f.reg.For("netbox_fake", 1)
}

func (fakeResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {}
func (fakeResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse)       {}
func (fakeResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {}
func (fakeResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {}

type fakeV0 struct {
	ID    types.Int64  `tfsdk:"id"`
	Label types.String `tfsdk:"label"`
	Port  types.String `tfsdk:"port"`
}

type fakeV1 struct {
	ID    types.Int64  `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Label types.String `tfsdk:"label"`
	Port  types.Int64  `tfsdk:"port"`
}

func fakeUpgraderFrom0() resource.StateUpgrader {
	return resource.StateUpgrader{
		PriorSchema: &schema.Schema{Attributes: map[string]schema.Attribute{
			"id":    schema.Int64Attribute{Computed: true},
			"label": schema.StringAttribute{Optional: true},
			"port":  schema.StringAttribute{Optional: true},
		}},
		StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
			var old fakeV0
			resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
			if resp.Diagnostics.HasError() {
				return
			}
			out := fakeV1{ID: old.ID, Name: old.Label, Label: old.Label, Port: types.Int64Null()}
			if !old.Port.IsNull() {
				n, err := strconv.ParseInt(old.Port.ValueString(), 10, 64)
				if err != nil {
					resp.Diagnostics.AddError("Invalid port in state", err.Error())
					return
				}
				out.Port = types.Int64Value(n)
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, out)...)
		},
	}
}

type fakeProvider struct{ reg *Registry }

func (fakeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "netbox"
}
func (fakeProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {}
func (fakeProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}
func (p fakeProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource { return fakeResource(p) }}
}
func (fakeProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }

var fakeV1Type = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"id": tftypes.Number, "name": tftypes.String, "label": tftypes.String, "port": tftypes.Number,
}}

// TestUpgradeStateEndToEnd drives UpgradeResourceState through the protocol
// 6 server, as Terraform does when it finds state of an older schema version.
func TestUpgradeStateEndToEnd(t *testing.T) {
	ctx := context.Background()
	reg := &Registry{}
	reg.Register("netbox_fake", 0, fakeUpgraderFrom0())
	if err := reg.Validate("netbox_fake", 1); err != nil {
		t.Fatal(err)
	}
	srv, err := providerserver.NewProtocol6WithError(fakeProvider{reg: reg})()
	if err != nil {
		t.Fatal(err)
	}
	upgradeTo := func(t *testing.T, version int64, raw string) map[string]tftypes.Value {
		t.Helper()
		resp, err := srv.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
			TypeName: "netbox_fake",
			Version:  version,
			RawState: &tfprotov6.RawState{JSON: []byte(raw)},
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range resp.Diagnostics {
			if d.Severity == tfprotov6.DiagnosticSeverityError {
				t.Fatalf("%s: %s", d.Summary, d.Detail)
			}
		}
		v, err := resp.UpgradedState.Unmarshal(fakeV1Type)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]tftypes.Value
		if err := v.As(&m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	wantString := func(t *testing.T, v tftypes.Value, want string) {
		t.Helper()
		var s string
		if err := v.As(&s); err != nil || s != want {
			t.Fatalf("got %v (%v), want %q", v, err, want)
		}
	}
	wantPort := func(t *testing.T, v tftypes.Value, want int64) {
		t.Helper()
		var n big.Float
		if err := v.As(&n); err != nil || n.Cmp(new(big.Float).SetInt64(want)) != 0 {
			t.Fatalf("port = %v (%v), want %d", v, err, want)
		}
	}

	t.Run("from version 0", func(t *testing.T) {
		m := upgradeTo(t, 0, `{"id":7,"label":"web","port":"443"}`)
		wantString(t, m["name"], "web")
		wantString(t, m["label"], "web")
		wantPort(t, m["port"], 443)
	})
	t.Run("current version passes through", func(t *testing.T) {
		m := upgradeTo(t, 1, `{"id":7,"name":"web","label":null,"port":8443}`)
		wantString(t, m["name"], "web")
		wantPort(t, m["port"], 8443)
		if !m["label"].IsNull() {
			t.Fatalf("label = %v, want null", m["label"])
		}
	})
	t.Run("upgrader error surfaces", func(t *testing.T) {
		resp, err := srv.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
			TypeName: "netbox_fake", Version: 0,
			RawState: &tfprotov6.RawState{JSON: []byte(`{"id":7,"label":null,"port":"https"}`)},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Diagnostics) == 0 || !strings.Contains(resp.Diagnostics[0].Summary, "Invalid port") {
			t.Fatalf("diagnostics = %v", resp.Diagnostics)
		}
	})
	t.Run("unregistered version fails", func(t *testing.T) {
		empty := fakeProvider{reg: &Registry{}}
		srv, err := providerserver.NewProtocol6WithError(empty)()
		if err != nil {
			t.Fatal(err)
		}
		resp, err := srv.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
			TypeName: "netbox_fake", Version: 0,
			RawState: &tfprotov6.RawState{JSON: []byte(`{"id":7,"label":null,"port":null}`)},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Diagnostics) == 0 || resp.Diagnostics[0].Severity != tfprotov6.DiagnosticSeverityError {
			t.Fatalf("expected an error, got %v", resp.Diagnostics)
		}
	})
}
