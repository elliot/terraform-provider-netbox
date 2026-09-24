package provider

import (
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// The registry lets generated packages contribute resources and data sources
// from their init() functions without this package importing them explicitly:
//
//	func init() { provider.RegisterResource(NewSiteResource) }
//
// The provider's Resources() and DataSources() methods return copies of the
// registered factories.

var (
	registryMu          sync.RWMutex
	resourceFactories   []func() resource.Resource
	dataSourceFactories []func() datasource.DataSource
)

// RegisterResource adds a resource factory to the provider.
func RegisterResource(f func() resource.Resource) {
	if f == nil {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	resourceFactories = append(resourceFactories, f)
}

// RegisterDataSource adds a data source factory to the provider.
func RegisterDataSource(f func() datasource.DataSource) {
	if f == nil {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	dataSourceFactories = append(dataSourceFactories, f)
}

// registeredResources returns a copy of the resource factories.
func registeredResources() []func() resource.Resource {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]func() resource.Resource, len(resourceFactories))
	copy(out, resourceFactories)
	return out
}

// registeredDataSources returns a copy of the data source factories.
func registeredDataSources() []func() datasource.DataSource {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]func() datasource.DataSource, len(dataSourceFactories))
	copy(out, dataSourceFactories)
	return out
}
