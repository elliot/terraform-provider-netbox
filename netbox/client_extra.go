// Package netbox is a Go client for the NetBox REST API, generated with
// openapi-generator from the NetBox 4.7 OpenAPI document using the same
// configuration as github.com/netbox-community/go-netbox. This file is
// hand-written and survives regeneration (see tools/client-gen/generate.sh).
package netbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultPageSize is the page size used by ListAll. NetBox's MAX_PAGE_SIZE
// defaults to 1000; 250 keeps individual responses small.
const DefaultPageSize int32 = 250

// NewClient returns an APIClient for the NetBox instance at serverURL
// (scheme://host[:port][/prefix], without the /api suffix). Authentication and
// user agent headers are expected to be injected by httpClient's transport; if
// token is non-empty a v2 "Authorization: Bearer" header is set as a default
// header instead.
func NewClient(serverURL string, httpClient *http.Client, token string) *APIClient {
	cfg := NewConfiguration()
	cfg.Servers = ServerConfigurations{{URL: strings.TrimSuffix(strings.TrimSuffix(serverURL, "/"), "/api")}}
	cfg.HTTPClient = httpClient
	cfg.Debug = false // never enable: dumps the Authorization header to the global logger
	if token != "" {
		cfg.AddDefaultHeader("Authorization", "Bearer "+token)
	}
	cfg.AddDefaultHeader("Accept-Language", "en-US")
	return NewAPIClient(cfg)
}

// Page is one page of a paginated NetBox list response.
type Page[T any] struct {
	Count   int32
	Results []T
}

// PageFunc fetches one page for the given limit/offset.
type PageFunc[T any] func(ctx context.Context, limit, offset int32) (Page[T], *http.Response, error)

// ListAll walks a paginated list endpoint until every object has been fetched.
//
//	sites, err := netbox.ListAll(ctx, func(ctx context.Context, limit, offset int32) (netbox.Page[netbox.Site], *http.Response, error) {
//	    p, res, err := c.DcimAPI.DcimSitesList(ctx).Limit(limit).Offset(offset).Execute()
//	    if err != nil { return netbox.Page[netbox.Site]{}, res, err }
//	    return netbox.Page[netbox.Site]{Count: p.GetCount(), Results: p.GetResults()}, res, nil
//	})
func ListAll[T any](ctx context.Context, fetch PageFunc[T]) ([]T, error) {
	return ListAllWithPageSize(ctx, DefaultPageSize, fetch)
}

// ListAllWithPageSize is ListAll with an explicit page size.
func ListAllWithPageSize[T any](ctx context.Context, pageSize int32, fetch PageFunc[T]) ([]T, error) {
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	var out []T
	var offset int32
	for {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		page, res, err := fetch(ctx, pageSize, offset)
		if err != nil {
			return out, WrapError(err, res)
		}
		out = append(out, page.Results...)
		offset += int32(len(page.Results))
		if len(page.Results) == 0 || offset >= page.Count {
			return out, nil
		}
	}
}

// APIError is a NetBox API error with the decoded response body.
type APIError struct {
	StatusCode int
	Status     string
	Body       []byte
	// Detail is the value of a top-level "detail" key, when present.
	Detail string
	// Fields holds field-level validation messages ("name": ["This field is required."]).
	Fields map[string][]string
}

func (e *APIError) Error() string {
	var b strings.Builder
	b.WriteString("NetBox API error")
	if e.Status != "" {
		b.WriteString(": " + e.Status)
	} else if e.StatusCode != 0 {
		fmt.Fprintf(&b, ": HTTP %d", e.StatusCode)
	}
	if e.Detail != "" {
		b.WriteString(": " + e.Detail)
	}
	if len(e.Fields) > 0 {
		keys := make([]string, 0, len(e.Fields))
		for k := range e.Fields {
			keys = append(keys, k)
		}
		sortStrings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+": "+strings.Join(e.Fields[k], "; "))
		}
		b.WriteString(": " + strings.Join(parts, ", "))
	} else if e.Detail == "" && len(e.Body) > 0 {
		body := strings.TrimSpace(string(e.Body))
		if len(body) > 512 {
			body = body[:512] + "..."
		}
		b.WriteString(": " + body)
	}
	return b.String()
}

// IsNotFound reports whether err is an APIError with HTTP 404.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// StatusCode returns the HTTP status carried by err, or 0.
func StatusCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

// WrapError converts the (err, *http.Response) pair returned by every generated
// Execute method into an *APIError carrying the decoded NetBox error body. Non-API
// errors (transport failures, context cancellation) are returned unchanged.
func WrapError(err error, res *http.Response) error {
	if err == nil {
		return nil
	}
	var already *APIError
	if errors.As(err, &already) {
		return err
	}
	var gen *GenericOpenAPIError
	var body []byte
	if errors.As(err, &gen) {
		body = gen.Body()
	} else if genv, ok := err.(GenericOpenAPIError); ok { //nolint:errorlint // value receiver variant
		body = genv.Body()
	} else if res == nil {
		return err
	}
	apiErr := &APIError{Body: body}
	if res != nil {
		apiErr.StatusCode = res.StatusCode
		apiErr.Status = res.Status
		if body == nil && res.Body != nil {
			body, _ = io.ReadAll(res.Body)
			apiErr.Body = body
		}
	}
	decodeErrorBody(apiErr)
	return apiErr
}

func decodeErrorBody(e *APIError) {
	if len(e.Body) == 0 {
		return
	}
	var generic map[string]json.RawMessage
	if err := json.Unmarshal(e.Body, &generic); err != nil {
		return
	}
	if raw, ok := generic["detail"]; ok {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			e.Detail = s
		}
		delete(generic, "detail")
	}
	for k, raw := range generic {
		var list []string
		if json.Unmarshal(raw, &list) == nil {
			if e.Fields == nil {
				e.Fields = map[string][]string{}
			}
			e.Fields[k] = list
			continue
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			if e.Fields == nil {
				e.Fields = map[string][]string{}
			}
			e.Fields[k] = []string{s}
			continue
		}
		// Nested structures (e.g. per-item bulk errors): keep raw JSON.
		if e.Fields == nil {
			e.Fields = map[string][]string{}
		}
		e.Fields[k] = []string{string(raw)}
	}
}

// PatchRaw sends a PATCH with an arbitrary JSON body to apiPath (for example
// "/api/dcim/sites/12/"). It exists for the few cases the typed models cannot
// express, such as clearing an untyped JSON field by sending an explicit null.
// The decoded response body is unmarshalled into out when out is non-nil.
func (c *APIClient) PatchRaw(ctx context.Context, apiPath string, body map[string]any, out any) error {
	return c.doRaw(ctx, http.MethodPatch, apiPath, body, out)
}

// GetRaw performs a GET against apiPath and unmarshals the JSON response into out.
func (c *APIClient) GetRaw(ctx context.Context, apiPath string, out any) error {
	return c.doRaw(ctx, http.MethodGet, apiPath, nil, out)
}

// PostRaw performs a POST against apiPath with a JSON body and unmarshals the
// response into out. Used for action endpoints such as available-ips.
func (c *APIClient) PostRaw(ctx context.Context, apiPath string, body any, out any) error {
	return c.doRaw(ctx, http.MethodPost, apiPath, body, out)
}

// DeleteRaw performs a DELETE against apiPath.
func (c *APIClient) DeleteRaw(ctx context.Context, apiPath string) error {
	return c.doRaw(ctx, http.MethodDelete, apiPath, nil, nil)
}

func (c *APIClient) doRaw(ctx context.Context, method, apiPath string, body any, out any) error {
	base, err := c.cfg.ServerURLWithContext(ctx, "")
	if err != nil {
		return err
	}
	if !strings.HasPrefix(apiPath, "/") {
		apiPath = "/" + apiPath
	}
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(base, "/")+apiPath, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.cfg.UserAgent != "" {
		req.Header.Set("User-Agent", c.cfg.UserAgent)
	}
	for k, v := range c.cfg.DefaultHeader {
		req.Header.Add(k, v)
	}
	res, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: res.StatusCode, Status: res.Status, Body: data}
		decodeErrorBody(apiErr)
		return apiErr
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

// Int32ID converts an int64 Terraform ID into the int32 the client expects.
func Int32ID(id int64) (int32, error) {
	if id < 0 || id > 1<<31-1 {
		return 0, fmt.Errorf("id %d out of range for NetBox (int32)", id)
	}
	return int32(id), nil
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
