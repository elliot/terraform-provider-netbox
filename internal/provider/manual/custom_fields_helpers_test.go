package manual_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type terraformState = terraform.State

func regexpMust(s string) *regexp.Regexp { return regexp.MustCompile(s) }

func snakeName(name string) string { return strings.ReplaceAll(name, "-", "_") }

// importID builds "<object_type>/<object_id>/<name>" from a custom field value resource in state.
func importID(s *terraform.State, address string) (string, error) {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return "", fmt.Errorf("%s not found in state", address)
	}
	a := rs.Primary.Attributes
	return fmt.Sprintf("%s/%s/%s", a["object_type"], a["object_id"], a["name"]), nil
}

// checkJSONFields asserts that the JSON object v contains the expected keys/values.
func checkJSONFields(v string, want map[string]any) error {
	var got map[string]any
	if err := json.Unmarshal([]byte(v), &got); err != nil {
		return fmt.Errorf("custom_fields is not JSON: %w", err)
	}
	for k, w := range want {
		if !reflect.DeepEqual(got[k], w) {
			return fmt.Errorf("custom_fields[%s] = %#v, want %#v (all: %s)", k, got[k], w, v)
		}
	}
	return nil
}
