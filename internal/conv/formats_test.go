package conv

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFormatValidators(t *testing.T) {
	for _, tc := range []struct {
		name  string
		v     validator.String
		value types.String
		ok    bool
	}{
		{"mac lower", MACAddress(), types.StringValue("02:00:5e:10:20:30"), true},
		{"mac upper", MACAddress(), types.StringValue("02:00:5E:10:20:30"), true},
		{"mac null", MACAddress(), types.StringNull(), true},
		{"mac unknown", MACAddress(), types.StringUnknown(), true},
		// NetBox accepts these but stores colon notation: a diff after apply.
		{"mac dashes", MACAddress(), types.StringValue("02-00-5e-10-20-30"), false},
		{"mac cisco dots", MACAddress(), types.StringValue("0200.5e10.2030"), false},
		{"mac bare", MACAddress(), types.StringValue("02005e102030"), false},
		{"mac short", MACAddress(), types.StringValue("02:00:5e:10:20"), false},
		{"mac eui64", MACAddress(), types.StringValue("02:00:5e:10:20:30:40:50"), false},
		{"mac not hex", MACAddress(), types.StringValue("02:00:5e:10:20:3g"), false},
		{"mac blank", MACAddress(), types.StringValue(""), false},
		{"wwn", WWN(), types.StringValue("50:01:43:80:12:34:56:78"), true},
		{"wwn lower", WWN(), types.StringValue("50:01:43:80:12:34:56:ab"), true},
		{"wwn eui48", WWN(), types.StringValue("50:01:43:80:12:34"), false},
		{"wwn dashes", WWN(), types.StringValue("50-01-43-80-12-34-56-78"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var resp validator.StringResponse
			tc.v.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("x"), ConfigValue: tc.value}, &resp)
			if got := !resp.Diagnostics.HasError(); got != tc.ok {
				t.Fatalf("valid = %v, want %v (%v)", got, tc.ok, resp.Diagnostics)
			}
		})
	}
}
