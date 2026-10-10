package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const testV1Token = "0123456789abcdef0123456789abcdef01234567"

func TestParseTokenVersion(t *testing.T) {
	cases := []struct {
		tok     string
		want    TokenVersion
		wantErr string
	}{
		{tok: "nbt_x.y", want: TokenV2},
		{tok: "nbt_abc123.s3cr3t.with.dots", want: TokenV2},
		{tok: testV1Token, want: TokenV1},
		{tok: strings.Repeat("Z", 40), want: TokenV1},
		{tok: "nbt_nodot", wantErr: "nbt_<key>.<secret>"},
		{tok: "nbt_.x", wantErr: "nbt_<key>.<secret>"},
		{tok: "nbt_x.", wantErr: "nbt_<key>.<secret>"},
		{tok: "abc", wantErr: "got 3 characters"},
		{tok: testV1Token + "8", wantErr: "got 41 characters"},
		{tok: "Token " + testV1Token, wantErr: "whitespace"},
		{tok: "Bearer nbt_x.y", wantErr: "whitespace"},
	}
	for _, tc := range cases {
		got, err := ParseTokenVersion(tc.tok)
		if tc.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("ParseTokenVersion(%q) error = %v, want one containing %q", tc.tok, err, tc.wantErr)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("ParseTokenVersion(%q) = %v, %v; want %v", tc.tok, got, err, tc.want)
		}
	}
}

func TestAuthorizationHeader(t *testing.T) {
	if got := AuthorizationHeader("nbt_x.y"); got != "Bearer nbt_x.y" {
		t.Errorf("v2: got %q", got)
	}
	if got := AuthorizationHeader(testV1Token); got != "Token "+testV1Token {
		t.Errorf("v1: got %q", got)
	}
}

func TestHeaderTransportV1Token(t *testing.T) {
	var (
		mu  sync.Mutex
		got string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got = r.Header.Get("Authorization")
		mu.Unlock()
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, func(cfg *Config) { cfg.Token = testV1Token })
	doRequest(t, c, http.MethodGet, srv.URL+"/api/status/", "")

	mu.Lock()
	defer mu.Unlock()
	if want := "Token " + testV1Token; got != want {
		t.Fatalf("Authorization = %q, want %q", got, want)
	}
}
