package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func fakeResponse(status int, method, rawURL string) *http.Response {
	u, _ := url.Parse(rawURL)
	return &http.Response{
		StatusCode: status,
		Request:    &http.Request{Method: method, URL: u},
	}
}

func TestParseAPIErrorFieldErrors(t *testing.T) {
	body := []byte(`{"name":["This field is required."],"slug":["Enter a valid slug.","Too long."]}`)
	e := ParseAPIError(fakeResponse(400, http.MethodPost, "https://nb/api/dcim/sites/"), body)

	if e.StatusCode != http.StatusBadRequest || e.Method != http.MethodPost || e.URL != "https://nb/api/dcim/sites/" {
		t.Fatalf("unexpected envelope: %+v", e)
	}
	if got := e.Fields["name"]; len(got) != 1 || got[0] != "This field is required." {
		t.Fatalf("name = %v", got)
	}
	if got := e.Fields["slug"]; len(got) != 2 || got[1] != "Too long." {
		t.Fatalf("slug = %v", got)
	}
	if e.Detail != "" {
		t.Fatalf("Detail = %q, want empty", e.Detail)
	}
	msg := e.Error()
	for _, want := range []string{"POST https://nb/api/dcim/sites/", "400 Bad Request", "name: This field is required.", "slug: Enter a valid slug.; Too long."} {
		if !strings.Contains(msg, want) {
			t.Errorf("Error() = %q, missing %q", msg, want)
		}
	}
}

func TestParseAPIErrorDetail(t *testing.T) {
	body := []byte(`{"detail":"Invalid token."}`)
	e := ParseAPIError(fakeResponse(403, http.MethodGet, "https://nb/api/status/"), body)
	if e.Detail != "Invalid token." {
		t.Fatalf("Detail = %q", e.Detail)
	}
	if len(e.Fields) != 0 {
		t.Fatalf("Fields = %v, want none", e.Fields)
	}
	if !strings.Contains(e.Error(), "403 Forbidden: Invalid token.") {
		t.Fatalf("Error() = %q", e.Error())
	}
}

func TestParseAPIErrorNonFieldErrors(t *testing.T) {
	body := []byte(`{"non_field_errors":["The fields site, name must make a unique set."]}`)
	e := ParseAPIError(fakeResponse(400, http.MethodPatch, "https://nb/api/dcim/racks/1/"), body)
	want := "The fields site, name must make a unique set."
	if e.Detail != want {
		t.Fatalf("Detail = %q, want %q", e.Detail, want)
	}
	if got := e.Fields["non_field_errors"]; len(got) != 1 || got[0] != want {
		t.Fatalf("non_field_errors = %v", got)
	}
}

func TestParseAPIErrorNestedAndNonJSON(t *testing.T) {
	nested := []byte(`{"custom_fields":{"vlan_group":["Object not found."]}}`)
	e := ParseAPIError(fakeResponse(400, http.MethodPost, "https://nb/api/x/"), nested)
	if got := e.Fields["custom_fields"]; len(got) != 1 || got[0] != "vlan_group: Object not found." {
		t.Fatalf("nested = %v", got)
	}

	html := []byte("<html>\n  <body>Bad Gateway</body>\n</html>")
	e = ParseAPIError(fakeResponse(502, http.MethodGet, "https://nb/api/x/"), html)
	if e.Detail != "<html> <body>Bad Gateway</body> </html>" {
		t.Fatalf("Detail = %q", e.Detail)
	}
	if string(e.Body) != string(html) {
		t.Fatal("raw body must be kept")
	}

	e = ParseAPIError(nil, nil)
	if e.StatusCode != 0 || e.Error() == "" {
		t.Fatalf("nil response: %+v", e)
	}
}

func TestIsNotFound(t *testing.T) {
	nf := ParseAPIError(fakeResponse(404, http.MethodGet, "https://nb/api/x/1/"), []byte(`{"detail":"Not found."}`))
	if !IsNotFound(nf) {
		t.Fatal("IsNotFound(404) = false")
	}
	if !IsNotFound(fmt.Errorf("wrapped: %w", nf)) {
		t.Fatal("IsNotFound must unwrap")
	}
	if IsNotFound(ParseAPIError(fakeResponse(400, "", ""), nil)) {
		t.Fatal("IsNotFound(400) = true")
	}
	if IsNotFound(errors.New("plain")) || IsNotFound(nil) {
		t.Fatal("IsNotFound on non-API errors must be false")
	}
	if !IsStatus(nf, 404) || IsStatus(nf, 403) {
		t.Fatal("IsStatus mismatch")
	}
}
