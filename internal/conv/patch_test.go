package conv

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/elliot/terraform-provider-netbox/netbox"
)

type recordingPatcher struct {
	path string
	body []byte
}

func (p *recordingPatcher) PatchRaw(_ context.Context, apiPath string, body map[string]any, _ any) error {
	p.path = apiPath
	b, err := json.Marshal(body)
	p.body = b
	return err
}

func TestPatchWithNulls(t *testing.T) {
	body := netbox.NewPatchedWritableDeviceRequest()
	body.SetDescription("x")
	var p recordingPatcher
	if err := PatchWithNulls(context.Background(), &p, "/api/dcim/devices/7/", body, []string{"local_context_data"}, nil); err != nil {
		t.Fatal(err)
	}
	if p.path != "/api/dcim/devices/7/" {
		t.Errorf("path = %q", p.path)
	}
	var got map[string]any
	if err := json.Unmarshal(p.body, &got); err != nil {
		t.Fatal(err)
	}
	if v, ok := got["local_context_data"]; !ok || v != nil {
		t.Errorf("local_context_data = %v (present %v), want explicit null", v, ok)
	}
	if got["description"] != "x" {
		t.Errorf("description = %v, want the typed fields kept", got["description"])
	}

	// The typed marshaller cannot express the null, which is why this exists.
	typed, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var typedMap map[string]any
	_ = json.Unmarshal(typed, &typedMap)
	if _, ok := typedMap["local_context_data"]; ok {
		t.Error("typed request now serialises local_context_data; PatchWithNulls may no longer be needed")
	}
}

type failingMap struct{}

func (failingMap) ToMap() (map[string]any, error) { return nil, errors.New("boom") }

func TestPatchWithNullsEncodingError(t *testing.T) {
	var p recordingPatcher
	if err := PatchWithNulls(context.Background(), &p, "/x/", failingMap{}, []string{"a"}, nil); err == nil {
		t.Fatal("expected the encoding error")
	}
	if p.path != "" {
		t.Error("nothing must be sent when encoding fails")
	}
}
