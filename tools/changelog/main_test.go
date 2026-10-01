package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const base = `# Changelog

Intro.

## [Unreleased]

### Security

- Hand-written security entry,
  wrapped onto a second line.

### Added

- Hand-written feature.

## [0.1.1] - 2026-09-24

### Fixed

- Old fix.

[Unreleased]: https://github.com/o/r/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/o/r/compare/v0.1.0...v0.1.1
`

const drafted = `
### Added

- ` + "`owner`" + `: Generated feature ([#11](https://github.com/o/r/pull/11))

### Breaking changes

- Generated breaking change

### Fixed

- Generated fix
`

func TestPrepare(t *testing.T) {
	got, err := prepare(base, "0.2.0", "2026-10-01", drafted)
	if err != nil {
		t.Fatal(err)
	}
	want := `# Changelog

Intro.

## [Unreleased]

## [0.2.0] - 2026-10-01

### Breaking changes

- Generated breaking change

### Added

- Hand-written feature.
- ` + "`owner`" + `: Generated feature ([#11](https://github.com/o/r/pull/11))

### Fixed

- Generated fix

### Security

- Hand-written security entry,
  wrapped onto a second line.

## [0.1.1] - 2026-09-24

### Fixed

- Old fix.

[Unreleased]: https://github.com/o/r/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/o/r/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/o/r/compare/v0.1.0...v0.1.1
`
	if got != want {
		t.Errorf("prepare mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}

	// The prepared file yields the new section as release notes.
	n, err := notes(got, "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(n, "### Breaking changes\n") || !strings.HasSuffix(n, "wrapped onto a second line.") {
		t.Errorf("notes = %q", n)
	}
}

func TestPrepareEmptyUnreleased(t *testing.T) {
	src := strings.Replace(base, base[strings.Index(base, "### Security"):strings.Index(base, "## [0.1.1]")], "", 1)
	got, err := prepare(src, "0.1.2", "2026-10-01", "### Fixed\n\n- Generated fix\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "## [Unreleased]\n\n## [0.1.2] - 2026-10-01\n\n### Fixed\n\n- Generated fix\n\n## [0.1.1]") {
		t.Errorf("unexpected output:\n%s", got)
	}
}

func TestPrepareErrors(t *testing.T) {
	empty := "# Changelog\n\n## [Unreleased]\n\n## [0.1.1] - 2026-09-24\n\n- x\n"
	for name, tc := range map[string]struct{ src, version, drafted, want string }{
		"nothing to release": {empty, "0.1.2", "\n", "nothing to release"},
		"duplicate version":  {base, "0.1.1", drafted, "already has"},
		"no unreleased":      {"# Changelog\n", "0.1.2", drafted, "no \"## [Unreleased]\""},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := prepare(tc.src, tc.version, "2026-10-01", tc.drafted)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestNotes(t *testing.T) {
	got, err := notes(base, "0.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "### Fixed\n\n- Old fix." {
		t.Errorf("notes = %q", got)
	}
	if _, err := notes(base, "0.3.0"); err == nil {
		t.Error("missing version: want error")
	}
	if _, err := notes("## [0.3.0] - 2026-10-01\n\n## [0.2.0]\n", "0.3.0"); err == nil {
		t.Error("empty section: want error")
	}
}

func TestRun(t *testing.T) {
	file := filepath.Join(t.TempDir(), "CHANGELOG.md")
	if err := os.WriteFile(file, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"prepare", "-file", file, "-version", "v0.2.0", "-date", "2026-10-01", "-entries", "-"},
		strings.NewReader(drafted), new(bytes.Buffer)); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"notes", "-file", file, "-version", "v0.2.0"}, nil, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Generated fix") {
		t.Errorf("notes output = %q", out.String())
	}
	if err := run([]string{"notes", "-file", file, "-version", "latest"}, nil, &out); err == nil {
		t.Error("invalid version: want error")
	}
}
