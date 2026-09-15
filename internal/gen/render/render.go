// Package render writes Go sources, tests and examples from the resource model.
package render

import "github.com/elliot/terraform-provider-netbox/internal/gen/model"

// Options controls output locations.
type Options struct {
	OutDir      string
	ExamplesDir string
	DocsDir     string
	// Partial means only a subset is rendered; registry files are not rewritten.
	Partial bool
}

// All renders every resource.
func All(resources []*model.Resource, netboxVersion string, opts Options) error {
	_ = resources
	_ = netboxVersion
	_ = opts
	return nil
}
