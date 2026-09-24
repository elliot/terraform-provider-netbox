package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// newTestClient builds a client against srv with fast retry settings; mutate
// tweaks the config before the client is constructed.
func newTestClient(t *testing.T, srv *httptest.Server, mutate func(*Config)) *http.Client {
	t.Helper()
	cfg := Defaults()
	cfg.ServerURL = srv.URL
	cfg.Token = "nbt_x.y"
	cfg.RetryWaitMin = time.Millisecond
	cfg.RetryWaitMax = 5 * time.Millisecond
	cfg.UserAgent = "test-agent/0.0"
	if mutate != nil {
		mutate(&cfg)
	}
	c, err := NewHTTPClient(cfg)
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	return c
}

func okHandler(hits *atomic.Int32) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if hits != nil {
			hits.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}
}

func doRequest(t *testing.T, c *http.Client, method, url string, body string) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, rdr)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp
}

func TestLimiterPacing(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(okHandler(&hits))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) {
		cfg.RequestsPerSecond = 5
		cfg.Burst = 1
	})

	start := time.Now()
	for i := 0; i < 3; i++ {
		doRequest(t, c, http.MethodGet, srv.URL+"/api/status/", "")
	}
	elapsed := time.Since(start)
	if elapsed < 400*time.Millisecond {
		t.Fatalf("3 requests at 5 rps took %s, want >= 400ms", elapsed)
	}
	if hits.Load() != 3 {
		t.Fatalf("hits = %d, want 3", hits.Load())
	}
}

func TestFixedRequestDelay(t *testing.T) {
	srv := httptest.NewServer(okHandler(nil))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) {
		cfg.RequestDelay = 100 * time.Millisecond
	})

	start := time.Now()
	doRequest(t, c, http.MethodGet, srv.URL+"/", "")
	doRequest(t, c, http.MethodGet, srv.URL+"/", "")
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Fatalf("2 requests with 100ms delay took %s, want >= 200ms", elapsed)
	}
}

func TestWriteDelayOnlyOnWrites(t *testing.T) {
	srv := httptest.NewServer(okHandler(nil))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) {
		cfg.WriteDelay = 200 * time.Millisecond
	})

	start := time.Now()
	doRequest(t, c, http.MethodGet, srv.URL+"/", "")
	if elapsed := time.Since(start); elapsed >= 150*time.Millisecond {
		t.Fatalf("GET took %s, write delay must not apply to reads", elapsed)
	}

	for _, m := range []string{http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete} {
		start = time.Now()
		doRequest(t, c, m, srv.URL+"/", `{"a":1}`)
		if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
			t.Fatalf("%s took %s, want >= 200ms write delay", m, elapsed)
		}
	}
}

func TestRetryAfter429(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"detail":"Request was throttled."}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) {
		cfg.RetryWaitMax = 10 * time.Second
	})

	start := time.Now()
	resp := doRequest(t, c, http.MethodGet, srv.URL+"/", "")
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if hits.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", hits.Load())
	}
	if elapsed < time.Second {
		t.Fatalf("Retry-After: 1 was not honoured, elapsed %s", elapsed)
	}
}

func TestRetryAfterIsCappedAtRetryWaitMax(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) == 1 {
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) {
		cfg.RetryWaitMax = 50 * time.Millisecond
	})

	start := time.Now()
	resp := doRequest(t, c, http.MethodGet, srv.URL+"/", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("Retry-After was not capped, elapsed %s", elapsed)
	}
}

func TestServerErrorRetriedThenSurfaced(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<html>boom</html>`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) {
		cfg.MaxRetries = 3
	})

	resp := doRequest(t, c, http.MethodGet, srv.URL+"/", "")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (last response must surface)", resp.StatusCode)
	}
	if hits.Load() != 4 {
		t.Fatalf("attempts = %d, want 4 (1 + MaxRetries)", hits.Load())
	}
}

func TestNotImplementedNotRetried(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusNotImplemented)
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil)
	resp := doRequest(t, c, http.MethodGet, srv.URL+"/", "")
	if resp.StatusCode != http.StatusNotImplemented || hits.Load() != 1 {
		t.Fatalf("status = %d attempts = %d, want 501 and 1", resp.StatusCode, hits.Load())
	}
}

func TestNotFoundNotRetried(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"Not found."}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil)
	resp := doRequest(t, c, http.MethodGet, srv.URL+"/api/dcim/sites/999/", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if hits.Load() != 1 {
		t.Fatalf("attempts = %d, want 1", hits.Load())
	}
}

func TestContextCancellationAbortsPendingWait(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(okHandler(&hits))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) {
		cfg.RequestsPerSecond = 1
		cfg.Burst = 1
	})

	// Consume the single token.
	doRequest(t, c, http.MethodGet, srv.URL+"/", "")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	resp, err := c.Do(req)
	elapsed := time.Since(start)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected an error after context cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("cancellation took %s, want well under the 1s limiter wait", elapsed)
	}
	if hits.Load() != 1 {
		t.Fatalf("server hits = %d, want 1 (cancelled request must not be sent)", hits.Load())
	}
}

func TestContextCancellationAbortsFixedDelay(t *testing.T) {
	srv := httptest.NewServer(okHandler(nil))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) {
		cfg.RequestDelay = 5 * time.Second
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/", nil)
	start := time.Now()
	resp, err := c.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected an error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("delay was not interrupted, elapsed %s", elapsed)
	}
}

func TestSerializeRequestsNeverOverlap(t *testing.T) {
	var inFlight, maxInFlight atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := inFlight.Add(1)
		for {
			cur := maxInFlight.Load()
			if n <= cur || maxInFlight.CompareAndSwap(cur, n) {
				break
			}
		}
		time.Sleep(15 * time.Millisecond)
		inFlight.Add(-1)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) {
		cfg.SerializeRequests = true
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/", nil)
			resp, err := c.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}()
	}
	wg.Wait()

	if maxInFlight.Load() != 1 {
		t.Fatalf("max in-flight = %d, want 1 with SerializeRequests", maxInFlight.Load())
	}
}

func TestHeadersAreSet(t *testing.T) {
	var got http.Header
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got = r.Header.Clone()
		mu.Unlock()
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) {
		cfg.Headers = map[string]string{
			"X-Extra":       "yes",
			"Authorization": "Token should-not-win",
		}
	})

	doRequest(t, c, http.MethodPost, srv.URL+"/api/dcim/sites/", `{"name":"x"}`)

	mu.Lock()
	defer mu.Unlock()
	if v := got.Get("Authorization"); v != "Bearer nbt_x.y" {
		t.Fatalf("Authorization = %q, want %q", v, "Bearer nbt_x.y")
	}
	if v := got.Get("Accept"); v != "application/json" {
		t.Fatalf("Accept = %q", v)
	}
	if v := got.Get("User-Agent"); v != "test-agent/0.0" {
		t.Fatalf("User-Agent = %q", v)
	}
	if v := got.Get("X-Extra"); v != "yes" {
		t.Fatalf("X-Extra = %q", v)
	}
	if v := got.Get("Content-Type"); v != "application/json" {
		t.Fatalf("Content-Type = %q, must be preserved from the caller", v)
	}
}

func TestPerAttemptTimeout(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) {
		cfg.RequestTimeout = 50 * time.Millisecond
		cfg.MaxRetries = 1
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/", nil)
	start := time.Now()
	resp, err := c.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected a timeout error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("timeout not applied, elapsed %s", elapsed)
	}
	if hits.Load() != 2 {
		t.Fatalf("attempts = %d, want 2 (timeouts are retried)", hits.Load())
	}
}

func TestBackoffAndCheckRetryHelpers(t *testing.T) {
	t.Run("retry-after seconds", func(t *testing.T) {
		resp := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": {"3"}}}
		if d := backoff(time.Second, 30*time.Second, 0, resp); d != 3*time.Second {
			t.Fatalf("backoff = %s, want 3s", d)
		}
	})
	t.Run("retry-after http-date", func(t *testing.T) {
		when := time.Now().Add(4 * time.Second).UTC().Format(http.TimeFormat)
		resp := &http.Response{StatusCode: http.StatusServiceUnavailable, Header: http.Header{"Retry-After": {when}}}
		d := backoff(time.Second, 30*time.Second, 0, resp)
		if d < 2*time.Second || d > 4*time.Second {
			t.Fatalf("backoff = %s, want ~3-4s", d)
		}
	})
	t.Run("retry-after ignored on 500", func(t *testing.T) {
		resp := &http.Response{StatusCode: http.StatusInternalServerError, Header: http.Header{"Retry-After": {"100"}}}
		if d := backoff(time.Second, 2*time.Second, 5, resp); d < time.Second || d > 2*time.Second {
			t.Fatalf("backoff = %s, want within [1s,2s]", d)
		}
	})
	t.Run("exponential bounds", func(t *testing.T) {
		for attempt := 0; attempt < 10; attempt++ {
			d := backoff(100*time.Millisecond, time.Second, attempt, nil)
			if d < 100*time.Millisecond || d > time.Second {
				t.Fatalf("attempt %d: backoff = %s out of [100ms,1s]", attempt, d)
			}
		}
	})
	t.Run("check retry", func(t *testing.T) {
		cases := map[int]bool{200: false, 400: false, 404: false, 429: true, 500: true, 501: false, 502: true, 503: true}
		for code, want := range cases {
			got, err := checkRetry(context.Background(), &http.Response{StatusCode: code}, nil)
			if err != nil || got != want {
				t.Fatalf("checkRetry(%d) = %v, %v; want %v", code, got, err, want)
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if got, err := checkRetry(ctx, &http.Response{StatusCode: http.StatusInternalServerError}, nil); got || !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled ctx: got %v, %v", got, err)
		}
	})
}

func TestNewRateLimiter(t *testing.T) {
	if l := NewRateLimiter(0, 10); l != nil {
		t.Fatal("rps 0 must disable the limiter")
	}
	l := NewRateLimiter(2, 0)
	if l == nil || l.Burst() != 1 || l.Limit() != 2 {
		t.Fatalf("unexpected limiter %+v", l)
	}
}

func TestConfigValidate(t *testing.T) {
	good := Defaults()
	good.ServerURL = "https://netbox.example.com"
	good.Token = "nbt_a.b"
	if err := good.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	bad := []func(*Config){
		func(c *Config) { c.ServerURL = "" },
		func(c *Config) { c.ServerURL = "ftp://x" },
		func(c *Config) { c.ServerURL = "https://" },
		func(c *Config) { c.Token = "" },
		func(c *Config) { c.RequestsPerSecond = -1 },
		func(c *Config) { c.RequestsPerSecond = 1; c.Burst = 0 },
		func(c *Config) { c.RequestDelay = -time.Second },
		func(c *Config) { c.MaxRetries = -1 },
		func(c *Config) { c.RetryWaitMin = 2 * time.Second; c.RetryWaitMax = time.Second },
		func(c *Config) { c.RequestTimeout = -1 },
		func(c *Config) { c.Headers = map[string]string{" ": "x"} },
	}
	for i, mut := range bad {
		c := good
		c.Headers = nil
		mut(&c)
		if err := c.Validate(); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}

	if _, err := NewHTTPClient(Config{}); err == nil {
		t.Fatal("NewHTTPClient must reject an empty config")
	}
}

func TestNormalizeServerURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://nb.example.com", "https://nb.example.com"},
		{"https://nb.example.com/", "https://nb.example.com"},
		{"https://nb.example.com/api", "https://nb.example.com"},
		{"https://nb.example.com/api/", "https://nb.example.com"},
		{" https://nb.example.com/nb/api/ ", "https://nb.example.com/nb"},
	}
	for _, tc := range cases {
		if got := NormalizeServerURL(tc.in); got != tc.want {
			t.Errorf("NormalizeServerURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
