package provider

import (
	"net/http"

	"github.com/elliot/terraform-provider-netbox/netbox"
)

// ProviderData is the handle handed to every resource and data source through
// resource.ConfigureRequest.ProviderData / datasource.ConfigureRequest.ProviderData.
type ProviderData struct {
	// HTTPClient is the fully decorated client (retry, pacing, auth, logging).
	// It can be used directly with net/http for endpoints the generated
	// client does not cover.
	HTTPClient *http.Client
	// ServerURL is the normalised base URL without a trailing slash or "/api".
	ServerURL string
	// Token is the v2 API token. Treat as sensitive; never log it.
	Token string
	// UserAgent is the User-Agent header sent with every request.
	UserAgent string
	// NetBoxVersion is the value of "netbox-version" reported by /api/status/,
	// or empty when the version check was skipped or failed.
	NetBoxVersion string
	// API is the generated NetBox API client (package netbox). Authentication,
	// pacing and retries are provided by HTTPClient's transport.
	API *netbox.APIClient
}
