// Package upgrade holds the hand-written state upgraders of generated
// resources.
//
// A generated resource whose override sets `schema_version: N` (N > 0)
// declares schema Version N and implements resource.ResourceWithUpgradeState
// by returning For("netbox_<name>", N). The migrations themselves are written
// by hand, one file per resource in this package, and registered from init():
//
//	func init() {
//		upgrade.Register("netbox_site", 0, resource.StateUpgrader{
//			PriorSchema:   &schema.Schema{ ... version 0 ... },
//			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) { ... },
//		})
//	}
//
// The framework calls exactly one upgrader per state: the one keyed by the
// version stored in the state. Each upgrader must therefore produce state of
// the current version directly; when the version is bumped again, every
// existing upgrader has to be extended to the new shape.
//
// Keeping the upgraders in this package (rather than next to the generated
// code) guarantees that they are linked into every binary that links a
// versioned resource, because the generated code imports this package.
package upgrade

import (
	"fmt"
	"sort"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Registry maps a Terraform resource type to its upgraders keyed by the prior
// schema version. The zero value is ready to use.
type Registry struct {
	mu        sync.RWMutex
	upgraders map[string]map[int64]resource.StateUpgrader
}

// Default is the registry the generated resources read.
var Default = &Registry{}

// Register adds the upgrader from fromVersion to the current schema version of
// resourceType to the Default registry. It panics on a duplicate or an invalid
// registration, which is a programming error caught by the first test run.
func Register(resourceType string, fromVersion int64, u resource.StateUpgrader) {
	Default.Register(resourceType, fromVersion, u)
}

// For returns the upgraders of resourceType from the Default registry; see
// Registry.For.
func For(resourceType string, current int64) map[int64]resource.StateUpgrader {
	return Default.For(resourceType, current)
}

// Register adds the upgrader from fromVersion to the current schema version of
// resourceType. It panics on a duplicate, a negative version or an upgrader
// without a function.
func (r *Registry) Register(resourceType string, fromVersion int64, u resource.StateUpgrader) {
	if resourceType == "" || fromVersion < 0 || u.StateUpgrader == nil {
		panic(fmt.Sprintf("upgrade: invalid registration %q from version %d", resourceType, fromVersion))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.upgraders == nil {
		r.upgraders = map[string]map[int64]resource.StateUpgrader{}
	}
	m := r.upgraders[resourceType]
	if m == nil {
		m = map[int64]resource.StateUpgrader{}
		r.upgraders[resourceType] = m
	}
	if _, dup := m[fromVersion]; dup {
		panic(fmt.Sprintf("upgrade: duplicate upgrader for %s from version %d", resourceType, fromVersion))
	}
	m[fromVersion] = u
}

// For returns a copy of the upgraders of resourceType whose prior version is
// below current, the resource's schema version. Missing versions are reported
// by Validate, not here: For runs inside UpgradeState, where a panic would
// crash the provider.
func (r *Registry) For(resourceType string, current int64) map[int64]resource.StateUpgrader {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[int64]resource.StateUpgrader{}
	for v, u := range r.upgraders[resourceType] {
		if v < current {
			out[v] = u
		}
	}
	return out
}

// Validate checks the upgraders of resourceType against its current schema
// version: every prior version 0..current-1 needs an upgrader, and none may
// start at or above the current version (left over after a version change).
func (r *Registry) Validate(resourceType string, current int64) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := r.upgraders[resourceType]
	var missing, stale []int64
	for v := int64(0); v < current; v++ {
		if _, ok := m[v]; !ok {
			missing = append(missing, v)
		}
	}
	for v := range m {
		if v >= current {
			stale = append(stale, v)
		}
	}
	sort.Slice(stale, func(i, j int) bool { return stale[i] < stale[j] })
	switch {
	case len(missing) > 0:
		return fmt.Errorf("%s has schema version %d but no state upgrader from version(s) %v", resourceType, current, missing)
	case len(stale) > 0:
		return fmt.Errorf("%s has schema version %d but state upgraders from version(s) %v, which are not older", resourceType, current, stale)
	}
	return nil
}

// Types returns the resource types with registered upgraders, sorted.
func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.upgraders))
	for t := range r.upgraders {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
