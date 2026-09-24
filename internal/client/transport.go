package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/time/rate"
)

// NewRateLimiter returns a token-bucket limiter allowing rps sustained
// requests per second with the given burst. It returns nil when rps <= 0,
// which the pacing transport treats as "no rate limiting".
func NewRateLimiter(rps float64, burst int) *rate.Limiter {
	if rps <= 0 {
		return nil
	}
	if burst < 1 {
		burst = 1
	}
	return rate.NewLimiter(rate.Limit(rps), burst)
}

// NewHTTPClient builds the fully decorated *http.Client for cfg. The returned
// client transparently retries, paces, authenticates and logs every request;
// callers use it exactly like a plain net/http client.
func NewHTTPClient(cfg Config) (*http.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid client configuration: %w", err)
	}

	base, err := newBaseTransport(cfg.InsecureSkipVerify)
	if err != nil {
		return nil, err
	}

	var rt http.RoundTripper = base
	rt = &loggingTransport{next: rt}
	rt = &headerTransport{
		next:      rt,
		token:     cfg.Token,
		userAgent: cfg.UserAgent,
		extra:     cloneHeaders(cfg.Headers),
	}
	rt = &timeoutTransport{next: rt, timeout: cfg.RequestTimeout}
	rt = newPacingTransport(rt, cfg)

	rc := retryablehttp.NewClient()
	rc.HTTPClient = &http.Client{Transport: rt}
	rc.RetryMax = cfg.MaxRetries
	rc.RetryWaitMin = cfg.RetryWaitMin
	rc.RetryWaitMax = cfg.RetryWaitMax
	rc.Logger = nil
	rc.CheckRetry = checkRetry
	rc.Backoff = backoff
	rc.ErrorHandler = errorHandler
	rc.RequestLogHook = recordAttempt

	return &http.Client{Transport: &retryTransport{client: rc}}, nil
}

// newBaseTransport clones http.DefaultTransport and optionally disables TLS
// verification.
func newBaseTransport(insecure bool) (*http.Transport, error) {
	dt, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("http.DefaultTransport is not an *http.Transport")
	}
	t := dt.Clone()
	if insecure {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		t.TLSClientConfig.InsecureSkipVerify = true //nolint:gosec // explicitly requested by the user
	}
	return t, nil
}

func cloneHeaders(h map[string]string) map[string]string {
	if len(h) == 0 {
		return nil
	}
	out := make(map[string]string, len(h))
	for k, v := range h {
		out[k] = v
	}
	return out
}

// ---------------------------------------------------------------------------
// retry layer
// ---------------------------------------------------------------------------

// attemptKey is the context key under which the current attempt number is
// stored for the logging transport.
type attemptKey struct{}

// retryTransport is the outermost RoundTripper. It hands the request to
// go-retryablehttp, which drives the inner chain once per attempt.
type retryTransport struct {
	client *retryablehttp.Client
}

// RoundTrip implements http.RoundTripper.
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	counter := new(atomic.Int32)
	ctx := context.WithValue(req.Context(), attemptKey{}, counter)

	rreq, err := retryablehttp.FromRequest(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	return t.client.Do(rreq)
}

// recordAttempt is wired as RequestLogHook and stores the 1-based attempt
// number in the request context so the logging transport can report it.
func recordAttempt(_ retryablehttp.Logger, req *http.Request, retryNumber int) {
	if c, ok := req.Context().Value(attemptKey{}).(*atomic.Int32); ok {
		c.Store(int32(retryNumber + 1)) //nolint:gosec // retry counts are tiny
	}
}

// attemptFromContext returns the attempt number recorded by recordAttempt, or
// 1 when the request did not pass through the retry layer.
func attemptFromContext(ctx context.Context) int {
	if c, ok := ctx.Value(attemptKey{}).(*atomic.Int32); ok {
		if n := c.Load(); n > 0 {
			return int(n)
		}
	}
	return 1
}

// checkRetry retries on transport errors, 429 and 5xx (except 501). It never
// retries once the context is done.
func checkRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil {
		// Delegate to the default policy so that permanent errors (bad
		// scheme, TLS verification failures, too many redirects) are not
		// retried.
		shouldRetry, _ := retryablehttp.DefaultRetryPolicy(ctx, resp, err)
		return shouldRetry, nil
	}
	if resp == nil {
		return true, nil
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return true, nil
	case resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented:
		return true, nil
	}
	return false, nil
}

// backoff honours Retry-After on 429/503 (capped at max) and otherwise uses
// exponential backoff with full jitter between min and the exponential bound.
func backoff(minWait, maxWait time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if resp != nil && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) {
		if d, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
			if d > maxWait {
				d = maxWait
			}
			if d < 0 {
				d = 0
			}
			return d
		}
	}

	if attemptNum > 30 {
		attemptNum = 30
	}
	upper := time.Duration(math.Pow(2, float64(attemptNum)) * float64(minWait))
	if upper <= 0 || upper > maxWait {
		upper = maxWait
	}
	if upper <= minWait {
		return minWait
	}
	return minWait + time.Duration(rand.Int64N(int64(upper-minWait))) //nolint:gosec // jitter, not security
}

// parseRetryAfter understands both "<seconds>" and HTTP-date forms.
func parseRetryAfter(v string) (time.Duration, bool) {
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.ParseFloat(v, 64); err == nil {
		if secs < 0 {
			secs = 0
		}
		return time.Duration(secs * float64(time.Second)), true
	}
	if t, err := http.ParseTime(v); err == nil {
		return time.Until(t), true
	}
	return 0, false
}

// errorHandler returns the last response unchanged so callers see the real
// status code after retries are exhausted. It never returns both a response
// and an error (net/http would log and discard the response).
func errorHandler(resp *http.Response, err error, numTries int) (*http.Response, error) {
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("netbox: giving up after %d attempt(s)", numTries)
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// pacing layer
// ---------------------------------------------------------------------------

// pacingTransport applies, in order: an optional serialization lock, a fixed
// delay (plus write delay for mutating methods) and a shared token-bucket
// limiter. All three honour the request context.
type pacingTransport struct {
	next         http.RoundTripper
	sem          chan struct{} // nil unless SerializeRequests; capacity 1
	requestDelay time.Duration
	writeDelay   time.Duration
	limiter      *rate.Limiter // nil when rate limiting is disabled
}

func newPacingTransport(next http.RoundTripper, cfg Config) *pacingTransport {
	p := &pacingTransport{
		next:         next,
		requestDelay: cfg.RequestDelay,
		writeDelay:   cfg.WriteDelay,
		limiter:      NewRateLimiter(cfg.RequestsPerSecond, cfg.Burst),
	}
	if cfg.SerializeRequests {
		p.sem = make(chan struct{}, 1)
	}
	return p
}

// RoundTrip implements http.RoundTripper.
func (p *pacingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	if p.sem != nil {
		select {
		case p.sem <- struct{}{}:
			defer func() { <-p.sem }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	delay := p.requestDelay
	if isWriteMethod(req.Method) {
		delay += p.writeDelay
	}
	if err := sleepContext(ctx, delay); err != nil {
		return nil, err
	}

	if p.limiter != nil {
		if err := p.limiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	return p.next.RoundTrip(req)
}

func isWriteMethod(m string) bool {
	switch m {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// sleepContext waits for d or until ctx is done, whichever comes first.
func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// ---------------------------------------------------------------------------
// per-attempt timeout layer
// ---------------------------------------------------------------------------

// timeoutTransport bounds a single attempt (headers and body) without
// counting the time spent waiting in the pacing layer above it.
type timeoutTransport struct {
	next    http.RoundTripper
	timeout time.Duration
}

// RoundTrip implements http.RoundTripper.
func (t *timeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.timeout <= 0 {
		return t.next.RoundTrip(req)
	}
	ctx, cancel := context.WithTimeout(req.Context(), t.timeout)
	resp, err := t.next.RoundTrip(req.WithContext(ctx))
	if err != nil {
		cancel()
		return nil, err
	}
	resp.Body = &cancelOnCloseBody{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

// cancelOnCloseBody releases the attempt context once the body is closed.
type cancelOnCloseBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelOnCloseBody) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

// ---------------------------------------------------------------------------
// header layer
// ---------------------------------------------------------------------------

// headerTransport adds authentication and default headers. It never touches
// Content-Type and never lets extra headers override Authorization.
type headerTransport struct {
	next      http.RoundTripper
	token     string
	userAgent string
	extra     map[string]string
}

// RoundTrip implements http.RoundTripper.
func (h *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	if r.Header.Get("Accept") == "" {
		r.Header.Set("Accept", "application/json")
	}
	if h.userAgent != "" {
		r.Header.Set("User-Agent", h.userAgent)
	}
	for k, v := range h.extra {
		if http.CanonicalHeaderKey(k) == "Authorization" {
			continue
		}
		r.Header.Set(k, v)
	}
	r.Header.Set("Authorization", "Bearer "+h.token)
	return h.next.RoundTrip(r)
}

// ---------------------------------------------------------------------------
// logging layer
// ---------------------------------------------------------------------------

// loggingTransport emits tflog debug/trace records. The Authorization header
// is never logged.
type loggingTransport struct {
	next http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (l *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	attempt := attemptFromContext(ctx)
	fields := map[string]any{
		"method":  req.Method,
		"url":     req.URL.String(),
		"attempt": attempt,
	}
	tflog.Trace(ctx, "netbox: sending request", fields)

	start := time.Now()
	resp, err := l.next.RoundTrip(req)
	fields["duration_ms"] = time.Since(start).Milliseconds()

	if err != nil {
		fields["error"] = err.Error()
		tflog.Debug(ctx, "netbox: request failed", fields)
		return nil, err
	}
	fields["status"] = resp.StatusCode
	tflog.Debug(ctx, "netbox: request completed", fields)
	return resp, nil
}
