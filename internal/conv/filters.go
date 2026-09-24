package conv

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// FilterName returns a validator for data source `filters[*].name` that
// rejects names not in allowed (sorted), so a typo fails `terraform validate`
// instead of silently widening the query (NetBox ignores unknown parameters
// and returns every object).
func FilterName(allowed []string) validator.String {
	return filterNameValidator{allowed: allowed}
}

type filterNameValidator struct {
	allowed []string
}

func (v filterNameValidator) Description(context.Context) string {
	return "must be a query parameter of the list endpoint"
}

func (v filterNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v filterNameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !Known(req.ConfigValue) {
		return
	}
	name := req.ConfigValue.ValueString()
	if !validFilter(v.allowed, name) {
		resp.Diagnostics.AddAttributeError(req.Path, "Unknown filter", unknownFilterDetail(name, v.allowed))
	}
}

func validFilter(allowed []string, name string) bool {
	if len(allowed) == 0 {
		return true
	}
	i := sort.SearchStrings(allowed, name)
	return i < len(allowed) && allowed[i] == name
}

// unknownFilterDetail explains an unknown filter name, suggesting the closest
// supported names and listing all of them only when nothing is close.
func unknownFilterDetail(name string, allowed []string) string {
	if s := SuggestFilters(name, allowed, 5); len(s) > 0 {
		return fmt.Sprintf("%q is not a supported filter. Did you mean %s?", name, quoteJoin(s, ", ", " or "))
	}
	return fmt.Sprintf("%q is not a supported filter. Supported filters: %s", name, strings.Join(allowed, ", "))
}

// SuggestFilters returns up to limit names from allowed that are close to
// name: within a small edit distance, or other lookups of the same field
// ("name__icontains" suggests "name__ic", "name__iew", ...). Closest first.
func SuggestFilters(name string, allowed []string, limit int) []string {
	base, _, hasLookup := strings.Cut(name, "__")
	maxDist := max(2, len(name)/4)
	type cand struct {
		name string
		dist int
	}
	var cands []cand
	for _, a := range allowed {
		d := levenshtein(name, a)
		sameField := hasLookup && (a == base || strings.HasPrefix(a, base+"__"))
		if d <= maxDist || sameField {
			cands = append(cands, cand{a, d})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].dist != cands[j].dist {
			return cands[i].dist < cands[j].dist
		}
		return cands[i].name < cands[j].name
	})
	out := make([]string, 0, min(limit, len(cands)))
	for i := 0; i < len(cands) && i < limit; i++ {
		out = append(out, cands[i].name)
	}
	return out
}

func quoteJoin(items []string, sep, last string) string {
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = fmt.Sprintf("%q", s)
	}
	if len(q) == 1 {
		return q[0]
	}
	return strings.Join(q[:len(q)-1], sep) + last + q[len(q)-1]
}

// levenshtein returns the edit distance between two ASCII strings.
func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
