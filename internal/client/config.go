// Package client builds the HTTP transport used by the NetBox provider.
//
// The transport chain (outer to inner) is:
//
//	retry (go-retryablehttp) -> pacing (serialize / fixed delay / token bucket)
//	-> per-attempt timeout -> headers (Bearer token, Accept, User-Agent)
//	-> logging (tflog, token redacted) -> base TLS transport
//
// Every wait in the chain honours the request context, so a cancelled
// Terraform operation never leaves a goroutine parked on a limiter or timer.
package client

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Config holds every knob that influences how requests reach NetBox.
type Config struct {
	// ServerURL is the base URL of the NetBox instance (scheme + host, optional
	// path prefix). A trailing slash or "/api" suffix is tolerated.
	ServerURL string
	// Token is a NetBox v2 API token ("nbt_<key>.<secret>"), sent as
	// "Authorization: Bearer <token>".
	Token string

	// RequestsPerSecond is the sustained request rate allowed by the shared
	// token-bucket limiter. Zero (or negative) disables rate limiting.
	RequestsPerSecond float64
	// Burst is the token-bucket capacity; only meaningful when
	// RequestsPerSecond > 0.
	Burst int
	// RequestDelay is a fixed artificial delay applied before every request.
	RequestDelay time.Duration
	// WriteDelay is an additional delay applied before POST, PUT, PATCH and
	// DELETE requests.
	WriteDelay time.Duration
	// SerializeRequests makes the client issue at most one request at a time.
	SerializeRequests bool

	// MaxRetries is the number of retries after the first attempt.
	MaxRetries int
	// RetryWaitMin and RetryWaitMax bound the backoff between retries.
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration
	// RequestTimeout bounds a single attempt (excluding pacing waits). Zero
	// disables the timeout.
	RequestTimeout time.Duration

	// InsecureSkipVerify disables TLS certificate verification.
	InsecureSkipVerify bool
	// Headers are extra headers added to every request. They cannot override
	// Authorization.
	Headers map[string]string
	// UserAgent is sent as the User-Agent header when non-empty.
	UserAgent string
}

// Default values applied by Defaults.
const (
	DefaultMaxRetries     = 4
	DefaultRetryWaitMin   = 1 * time.Second
	DefaultRetryWaitMax   = 30 * time.Second
	DefaultRequestTimeout = 60 * time.Second
	DefaultBurst          = 10
)

// Defaults returns a Config populated with the provider's default values.
// ServerURL and Token are left empty.
func Defaults() Config {
	return Config{
		Burst:          DefaultBurst,
		MaxRetries:     DefaultMaxRetries,
		RetryWaitMin:   DefaultRetryWaitMin,
		RetryWaitMax:   DefaultRetryWaitMax,
		RequestTimeout: DefaultRequestTimeout,
	}
}

// Validate reports the first configuration problem it finds.
func (c Config) Validate() error {
	if strings.TrimSpace(c.ServerURL) == "" {
		return errors.New("server URL must not be empty")
	}
	u, err := url.Parse(c.ServerURL)
	if err != nil {
		return fmt.Errorf("server URL %q is not a valid URL: %w", c.ServerURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("server URL %q must use the http or https scheme", c.ServerURL)
	}
	if u.Host == "" {
		return fmt.Errorf("server URL %q has no host", c.ServerURL)
	}
	if c.Token == "" {
		return errors.New("API token must not be empty")
	}
	if c.RequestsPerSecond < 0 {
		return errors.New("requests per second must not be negative")
	}
	if c.RequestsPerSecond > 0 && c.Burst < 1 {
		return errors.New("request burst must be at least 1 when rate limiting is enabled")
	}
	if c.RequestDelay < 0 || c.WriteDelay < 0 {
		return errors.New("request and write delays must not be negative")
	}
	if c.MaxRetries < 0 {
		return errors.New("max retries must not be negative")
	}
	if c.RetryWaitMin < 0 || c.RetryWaitMax < 0 {
		return errors.New("retry wait bounds must not be negative")
	}
	if c.RetryWaitMax < c.RetryWaitMin {
		return errors.New("retry wait max must be greater than or equal to retry wait min")
	}
	if c.RequestTimeout < 0 {
		return errors.New("request timeout must not be negative")
	}
	for k := range c.Headers {
		if strings.TrimSpace(k) == "" {
			return errors.New("header names must not be empty")
		}
	}
	return nil
}

// NormalizeServerURL strips whitespace, a trailing "/api" segment and trailing
// slashes so that "<url>/api/..." paths can be appended safely.
func NormalizeServerURL(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimRight(s, "/")
	s = strings.TrimSuffix(s, "/api")
	return strings.TrimRight(s, "/")
}
