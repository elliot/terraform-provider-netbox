package conv

import (
	"context"
	"fmt"
)

// RequestMap is implemented by the generated request models (ToMap is what
// their MarshalJSON serialises).
type RequestMap interface {
	ToMap() (map[string]any, error)
}

// RawPatcher sends a PATCH with an arbitrary JSON body (netbox.APIClient).
type RawPatcher interface {
	PatchRaw(ctx context.Context, apiPath string, body map[string]any, out any) error
}

// PatchWithNulls sends body as a PATCH to apiPath with each property in nulls
// set to an explicit JSON null. The generated models drop nil untyped JSON
// fields (omitempty), so clearing local_context_data and friends needs the raw
// request; everything else in the body is serialised exactly as the typed call
// would.
func PatchWithNulls(ctx context.Context, c RawPatcher, apiPath string, body RequestMap, nulls []string, out any) error {
	m, err := body.ToMap()
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}
	for _, k := range nulls {
		m[k] = nil
	}
	return c.PatchRaw(ctx, apiPath, m, out)
}
