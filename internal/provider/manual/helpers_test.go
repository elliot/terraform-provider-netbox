package manual_test

import (
	"fmt"
	"testing"

	"github.com/elliot/terraform-provider-netbox/internal/acctest"
)

// hash derives a small stable number from a test name so that fixtures on a
// shared NetBox (prefixes, ASN ranges) rarely collide with leftovers.
func hash(name string) int {
	h := 0
	for _, c := range name {
		h = (h*31 + int(c)) % 1000003
	}
	return h
}

// octet returns a third octet in 1..200 for the given test name.
func octet(name string) int { return 1 + hash(name)%200 }

// render substitutes the printf arguments into tmpl and then renders it with
// the acceptance test template ({{.Name}}).
func render(t *testing.T, tmpl, name string, args ...any) string {
	t.Helper()
	return acctest.Render(t, fmt.Sprintf(tmpl, args...), name)
}
