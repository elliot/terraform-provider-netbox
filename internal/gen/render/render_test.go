package render

import "testing"

func TestHeaderVersion(t *testing.T) {
	for in, want := range map[string]string{
		"4.7.2 (4.7)": "4.7.x",
		"4.7.0":       "4.7.x",
		"4.10.13":     "4.10.x",
		"5":           "5",
	} {
		if got := headerVersion(in); got != want {
			t.Errorf("headerVersion(%q) = %q, want %q", in, got, want)
		}
	}
}
