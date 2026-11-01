package overrides

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMergesEvolutionKeys(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("a.yaml", "resources:\n  sites:\n    deprecated: \"old\"\n    schema_version: 1\n    attributes:\n      x: { deprecated: true }\n      y: { deprecated: false }\n")
	write("b.yaml", "resources:\n  sites:\n    schema_version: 2\n")
	ov, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := ov["sites"]
	if s.SchemaVersion != 2 || s.Deprecated.Message != "old" {
		t.Fatalf("merged = %+v", s)
	}
	if got := s.Attributes["x"].Deprecated.Resolve(false, "def"); got != "def" {
		t.Errorf("deprecated: true resolves to %q", got)
	}
	if got := s.Attributes["y"].Deprecated.Resolve(true, "def"); got != "" {
		t.Errorf("deprecated: false resolves to %q", got)
	}
	if got := (Deprecation{}).Resolve(true, "def"); got != "def" {
		t.Errorf("unset with spec flag resolves to %q", got)
	}
	if got := (Deprecation{}).Resolve(false, "def"); got != "" {
		t.Errorf("unset without spec flag resolves to %q", got)
	}
}
