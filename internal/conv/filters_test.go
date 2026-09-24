package conv

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// A slice of /api/dcim/sites/ parameters, sorted like the generated lists.
var siteFilters = []string{
	"asn", "asn_id", "description", "description__ic", "id", "name", "name__empty", "name__ic", "name__ie",
	"name__iew", "name__isw", "name__n", "q", "region", "region_id", "slug", "status", "tag", "tenant", "tenant_id",
}

func TestFilterNameValidator(t *testing.T) {
	v := FilterName(siteFilters)
	for _, tc := range []struct {
		name    string
		value   types.String
		wantErr string
	}{
		{"known", types.StringValue("tenant_id"), ""},
		{"null", types.StringNull(), ""},
		{"unknown value", types.StringUnknown(), ""},
		{"typo", types.StringValue("tenat_id"), `"tenat_id" is not a supported filter. Did you mean "tenant_id"?`},
		{"wrong lookup", types.StringValue("name__icontains"), `Did you mean "name__ic", "name__empty", "name__ie", "name__iew" or "name__isw"?`},
		{"nothing close", types.StringValue("serial_number"), "Supported filters: asn, asn_id,"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("filters").AtListIndex(0).AtName("name"), ConfigValue: tc.value}
			var resp validator.StringResponse
			v.ValidateString(context.Background(), req, &resp)
			if tc.wantErr == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected error: %v", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected an error")
			}
			if got := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(got, tc.wantErr) {
				t.Fatalf("detail %q does not contain %q", got, tc.wantErr)
			}
		})
	}
}

func TestFilterNameValidatorEmptyAllowList(t *testing.T) {
	var resp validator.StringResponse
	FilterName(nil).ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue("anything")}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("an empty allow list accepts everything: %v", resp.Diagnostics)
	}
}

func TestSuggestFilters(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"stauts", []string{"status"}},
		{"regionid", []string{"region_id", "region"}},
		{"dscription__ic", []string{"description__ic"}},
		{"xyzzy_plugh", []string{}},
	} {
		if got := SuggestFilters(tc.in, siteFilters, 3); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("SuggestFilters(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"", "", 0}, {"abc", "", 3}, {"", "abc", 3}, {"kitten", "sitting", 3}, {"tenant_id", "tenat_id", 1},
	} {
		if got := levenshtein(tc.a, tc.b); got != tc.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
