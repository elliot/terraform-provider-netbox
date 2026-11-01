package render

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestAppPackagesOnDisk(t *testing.T) {
	dir := t.TempDir()
	write := func(rel string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("dcim/site_resource.go")
	write("ipam/prefix_resource.go")
	write("all/all.go")
	write("objecttypes/objecttypes.go")
	if err := os.MkdirAll(filepath.Join(dir, "empty"), 0o750); err != nil {
		t.Fatal(err)
	}

	got := appPackagesOnDisk(dir)
	if want := []string{"dcim", "ipam"}; !slices.Equal(got, want) {
		t.Fatalf("appPackagesOnDisk = %v, want %v", got, want)
	}
}
