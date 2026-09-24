package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// maxBodyInError caps the amount of raw body reproduced in an error message.
const maxBodyInError = 512

// APIError describes a non-2xx response from NetBox.
type APIError struct {
	StatusCode int
	Method     string
	URL        string
	// Body is the raw response body.
	Body []byte
	// Fields holds per-field validation messages from a DRF 400 response
	// (including "non_field_errors").
	Fields map[string][]string
	// Detail is the "detail" message, or a best-effort summary when the body
	// is not a JSON object.
	Detail string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	var b strings.Builder
	b.WriteString("netbox: ")
	if e.Method != "" || e.URL != "" {
		fmt.Fprintf(&b, "%s %s ", e.Method, e.URL)
	}
	fmt.Fprintf(&b, "returned %d %s", e.StatusCode, http.StatusText(e.StatusCode))

	var parts []string
	if e.Detail != "" {
		parts = append(parts, e.Detail)
	}
	keys := make([]string, 0, len(e.Fields))
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, strings.Join(e.Fields[k], "; ")))
	}
	if len(parts) > 0 {
		b.WriteString(": ")
		b.WriteString(strings.Join(parts, "; "))
	}
	return b.String()
}

// ParseAPIError builds an APIError from a response and its (already read)
// body. It understands the three shapes NetBox/DRF produce:
//
//	{"field": ["message", ...]}
//	{"detail": "message"}
//	{"non_field_errors": ["message", ...]}
//
// Anything else (HTML error pages, arrays, plain text) is summarised in Detail.
func ParseAPIError(resp *http.Response, body []byte) *APIError {
	e := &APIError{Body: body}
	if resp != nil {
		e.StatusCode = resp.StatusCode
		if resp.Request != nil {
			e.Method = resp.Request.Method
			if resp.Request.URL != nil {
				e.URL = resp.Request.URL.String()
			}
		}
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return e
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err == nil {
		for k, raw := range obj {
			msgs := rawToStrings(raw)
			if k == "detail" {
				e.Detail = strings.Join(msgs, "; ")
				continue
			}
			if e.Fields == nil {
				e.Fields = make(map[string][]string)
			}
			e.Fields[k] = msgs
		}
		if e.Detail == "" {
			if nfe, ok := e.Fields["non_field_errors"]; ok {
				e.Detail = strings.Join(nfe, "; ")
			}
		}
		return e
	}

	// Not a JSON object: keep a short, single-line excerpt.
	if len(trimmed) > maxBodyInError {
		trimmed = trimmed[:maxBodyInError] + "..."
	}
	e.Detail = strings.Join(strings.Fields(trimmed), " ")
	return e
}

// rawToStrings flattens a JSON value into human-readable strings.
func rawToStrings(raw json.RawMessage) []string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []string{s}
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			out = append(out, rawToStrings(item)...)
		}
		return out
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err == nil {
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]string, 0, len(keys))
		for _, k := range keys {
			out = append(out, fmt.Sprintf("%s: %s", k, strings.Join(rawToStrings(obj[k]), "; ")))
		}
		return out
	}
	return []string{string(raw)}
}

// IsNotFound reports whether err is an APIError with status 404.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsStatus reports whether err is an APIError with the given status code.
func IsStatus(err error, status int) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == status
	}
	return false
}
