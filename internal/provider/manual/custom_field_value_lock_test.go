package manual

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/elliot/terraform-provider-netbox/netbox"
)

// TestCustomFieldValueWrite_parallelFieldsOnOneObject reproduces the lost
// update behind the flaky TestAccCustomFieldValue_basic: NetBox merges a
// custom_fields PATCH into a snapshot of the object and saves the whole set, so
// overlapping PATCHes of different fields on one object drop all but one. The
// fake server below behaves the same way (with a delay that makes the overlap
// certain), and the resource must still end up with every field.
func TestCustomFieldValueWrite_parallelFieldsOnOneObject(t *testing.T) {
	var (
		mu     sync.Mutex
		stored = map[string]any{}
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var in struct {
			CustomFields map[string]any `json:"custom_fields"`
		}
		if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		merged := maps.Clone(stored)
		mu.Unlock()
		maps.Copy(merged, in.CustomFields)
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		stored = merged
		out := maps.Clone(stored)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"custom_fields": out})
	}))
	defer srv.Close()

	r := &CustomFieldValueResource{client: netbox.NewClient(srv.URL, srv.Client(), "")}

	const fields = 4
	var wg sync.WaitGroup
	for i := range fields {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := &CustomFieldValueModel{
				ObjectType: types.StringValue("dcim.site"),
				ObjectId:   types.Int64Value(1),
				Name:       types.StringValue(fmt.Sprintf("field_%d", i)),
				Value:      types.DynamicValue(types.StringValue(fmt.Sprintf("value-%d", i))),
			}
			var diags diag.Diagnostics
			r.write(t.Context(), m, false, &diags)
			if diags.HasError() {
				t.Errorf("field_%d: %v", i, diags)
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	for i := range fields {
		name, want := fmt.Sprintf("field_%d", i), fmt.Sprintf("value-%d", i)
		if got := stored[name]; got != want {
			t.Errorf("%s = %v, want %q (stored: %v)", name, got, want, stored)
		}
	}
}

func TestKeyedMutex_separatesKeys(t *testing.T) {
	var k keyedMutex
	unlockA := k.lock("a")
	// A different key must not block while "a" is held.
	done := make(chan struct{})
	go func() {
		k.lock("b")()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("lock on a different key blocked")
	}
	unlockA()
	// The same key can be taken again once released.
	k.lock("a")()
}
