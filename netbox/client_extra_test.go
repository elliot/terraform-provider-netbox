package netbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestListAllPaginates(t *testing.T) {
	total := 7
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		calls++
		if got := r.Header.Get("Authorization"); got != "Bearer nbt_k.t" {
			t.Errorf("Authorization = %q", got)
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var results []map[string]any
		for i := offset; i < total && i < offset+limit; i++ {
			results = append(results, map[string]any{"id": i + 1, "name": fmt.Sprintf("s%d", i+1), "slug": fmt.Sprintf("s%d", i+1)})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"count": total, "results": results})
	}))
	defer srv.Close()

	c := NewClient(srv.URL+"/", srv.Client(), "nbt_k.t")
	sites, err := ListAllWithPageSize(context.Background(), 3, func(ctx context.Context, limit, offset int32) (Page[Site], *http.Response, error) {
		p, res, err := c.DcimAPI.DcimSitesList(ctx).Limit(limit).Offset(offset).Execute()
		if err != nil {
			return Page[Site]{}, res, err
		}
		return Page[Site]{Count: p.GetCount(), Results: p.GetResults()}, res, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != total {
		t.Fatalf("got %d sites, want %d", len(sites), total)
	}
	if calls != 3 {
		t.Fatalf("got %d page requests, want 3", calls)
	}
	if sites[6].GetName() != "s7" {
		t.Fatalf("unexpected last site %+v", sites[6])
	}
}

func TestWrapErrorDecodesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"name":["This field is required."],"slug":["Enter a valid slug."]}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, srv.Client(), "")
	_, res, err := c.DcimAPI.DcimSitesCreate(context.Background()).WritableSiteRequest(*NewWritableSiteRequest("", "")).Execute()
	err = WrapError(err, res)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T %v", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Fields["name"][0] != "This field is required." {
		t.Fatalf("unexpected %+v", apiErr)
	}
	want := "NetBox API error: 400 Bad Request: name: This field is required., slug: Enter a valid slug."
	if apiErr.Error() != want {
		t.Fatalf("Error() = %q", apiErr.Error())
	}
}

func TestWrapErrorNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"Not found."}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, srv.Client(), "")
	_, res, err := c.DcimAPI.DcimSitesRetrieve(context.Background(), 99).Execute()
	err = WrapError(err, res)
	if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if StatusCode(err) != 404 {
		t.Fatalf("StatusCode = %d", StatusCode(err))
	}
}

func TestPatchRaw(t *testing.T) {
	var gotBody map[string]any
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id": 5, "name": "x"}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, srv.Client(), "")
	var out Site
	if err := c.PatchRaw(context.Background(), "/api/dcim/sites/5/", map[string]any{"local_context_data": nil}, &out); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/api/dcim/sites/5/" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	if v, ok := gotBody["local_context_data"]; !ok || v != nil {
		t.Fatalf("null not sent: %v", gotBody)
	}
	if out.Id != 5 {
		t.Fatalf("out = %+v", out)
	}
}

func TestChoiceStringDecodesUnknownValue(t *testing.T) {
	var s Site
	if err := json.Unmarshal([]byte(`{"id":1,"status":{"value":"brand-new-status","label":"Brand New"},"description":""}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.Status.Value.Get() == nil || *s.Status.Value.Get() != "brand-new-status" {
		t.Fatalf("status = %+v", s.Status)
	}
}

func TestInt32ID(t *testing.T) {
	if _, err := Int32ID(1 << 40); err == nil {
		t.Fatal("expected range error")
	}
	if v, err := Int32ID(42); err != nil || v != 42 {
		t.Fatal("unexpected", v, err)
	}
}
