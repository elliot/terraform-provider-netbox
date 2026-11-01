// Command schema-diff compares two `terraform providers schema -json`
// documents (usually the latest provider release and the working tree) and
// classifies every difference, so a pull request cannot silently remove or
// reshape something users depend on. `make schema-diff` and the Schema diff
// workflow drive it; see docs/DEVELOPMENT.md.
//
//	schema-diff -old base.json -new current.json [-old-label v0.5.0] [-new-label HEAD] \
//	    [-provider registry.terraform.io/elliot/netbox] [-out report.md] [-allow-breaking]
//	schema-diff latest elliot/netbox     # prints the latest release on the registry
//
// The Markdown report goes to stdout (or -out) and is meant for
// $GITHUB_STEP_SUMMARY. The exit status is 1 when a breaking change is found
// and -allow-breaking is not set, 2 on usage or input errors, 0 otherwise.
//
// # Classification
//
// Breaking, because an existing configuration can stop validating, plan an
// unexpected change, or lose a value it references:
//   - a resource, data source, ephemeral resource, list resource, action,
//     function or resource identity is removed;
//   - an attribute or nested block is removed;
//   - an attribute's type changes. This includes list <-> set (indexing a set
//     fails), changing a nested attribute's nesting mode, and any change in a
//     nested attribute's object type;
//   - a new required attribute, required nested block or identity attribute
//     required for import;
//   - optional or computed-only -> required;
//   - required, optional or optional+computed -> computed-only;
//   - optional+computed -> optional-only: an omitted value is null instead of
//     the server's value, so every existing resource that omits it plans a
//     change (in a data source the value is no longer filled in);
//   - computed-only -> optional-only, for the same reason;
//   - an attribute becomes write-only (no longer readable or in state);
//   - tighter nested block/attribute item limits, a lower schema version,
//     function parameter or return type changes.
//
// Additive: new types, new optional or computed attributes, new optional
// nested blocks, required -> optional, computed-only -> optional+computed,
// looser item limits.
//
// Notice (listed, never fails): sensitive added or removed (note that root
// module outputs exposing a newly sensitive value need `sensitive = true`),
// deprecation added or removed, optional -> optional+computed, a higher schema
// version, a switch between a plain object type and a nested attribute with
// the same value type, write-only removed. Below a computed-only nested
// attribute, flag changes are ignored because nothing there can be set.
//
// Descriptions are not classified; the report only counts the attributes
// whose description changed, because allowed values (validators) are not part
// of the schema and the generator lists them in descriptions.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "latest" {
		if err := latestCmd(os.Args[2:], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "schema-diff latest:", err)
			os.Exit(2)
		}
		return
	}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("schema-diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		oldPath       = fs.String("old", "", "schema JSON of the base (released) provider")
		newPath       = fs.String("new", "", "schema JSON of the candidate provider")
		oldLabel      = fs.String("old-label", "", "name of the base in the report (default: the -old path)")
		newLabel      = fs.String("new-label", "", "name of the candidate in the report (default: the -new path)")
		provider      = fs.String("provider", "", "provider address in both documents (default: the only one)")
		out           = fs.String("out", "", "write the Markdown report to this file instead of stdout")
		allowBreaking = fs.Bool("allow-breaking", false, "exit 0 even when breaking changes are found")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *oldPath == "" || *newPath == "" || fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: schema-diff -old base.json -new current.json [flags]")
		fs.PrintDefaults()
		return 2
	}
	if *oldLabel == "" {
		*oldLabel = *oldPath
	}
	if *newLabel == "" {
		*newLabel = *newPath
	}

	oldSchema, addr, err := loadProvider(*oldPath, *provider)
	if err != nil {
		fmt.Fprintf(stderr, "schema-diff: %s: %v\n", *oldPath, err)
		return 2
	}
	newSchema, _, err := loadProvider(*newPath, addr)
	if err != nil {
		fmt.Fprintf(stderr, "schema-diff: %s: %v\n", *newPath, err)
		return 2
	}

	result := Diff(oldSchema, newSchema)
	report := Report(result, *oldLabel, *newLabel, *allowBreaking)
	if *out != "" {
		if err := os.WriteFile(*out, []byte(report), 0o644); err != nil { //nolint:gosec // report is not secret
			fmt.Fprintln(stderr, "schema-diff:", err)
			return 2
		}
	} else if _, err := io.WriteString(stdout, report); err != nil {
		return 2
	}
	if count(result.Changes, Breaking) > 0 && !*allowBreaking {
		return 1
	}
	return 0
}

func count(changes []Change, sev Severity) int {
	n := 0
	for _, c := range changes {
		if c.Severity == sev {
			n++
		}
	}
	return n
}

// collapseAfter is the number of rows above which a section is folded into a
// <details> element, so a large additive diff does not bury breaking changes.
const collapseAfter = 15

// Report renders the result as Markdown.
func Report(result Result, oldLabel, newLabel string, allowBreaking bool) string {
	changes := result.Changes
	var b strings.Builder
	fmt.Fprintf(&b, "## Provider schema diff: %s -> %s\n\n", oldLabel, newLabel)
	nb, na, nn := count(changes, Breaking), count(changes, Additive), count(changes, Notice)
	fmt.Fprintf(&b, "Breaking: **%d**, additive: **%d**, notices: **%d**.\n\n", nb, na, nn)
	if result.DescriptionChanges > 0 {
		fmt.Fprintf(&b, "Descriptions changed on %d attributes. They are not classified, but allowed values (validators) "+
			"are not part of the schema and only show up there; check the docs diff for removed values.\n\n", result.DescriptionChanges)
	}
	switch {
	case len(changes) == 0:
		b.WriteString("No schema changes.\n")
		return b.String()
	case nb == 0:
		b.WriteString("No breaking changes.\n\n")
	case allowBreaking:
		b.WriteString("Breaking changes found and allowed.\n\n")
	default:
		b.WriteString("Breaking changes found: existing configurations may fail or plan unexpected changes.\n\n")
	}
	section(&b, "Breaking changes", changes, Breaking, false)
	section(&b, "Additive changes", changes, Additive, true)
	section(&b, "Notices", changes, Notice, true)
	return b.String()
}

func section(b *strings.Builder, title string, changes []Change, sev Severity, collapsible bool) {
	var rows []Change
	for _, c := range changes {
		if c.Severity == sev {
			rows = append(rows, c)
		}
	}
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(b, "### %s\n\n", title)
	folded := collapsible && len(rows) > collapseAfter
	if folded {
		fmt.Fprintf(b, "<details><summary>%d %s</summary>\n\n", len(rows), strings.ToLower(title))
	}
	b.WriteString("| Kind | Name | Attribute | Change |\n|---|---|---|---|\n")
	for _, c := range rows {
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", c.Kind, code(c.Name), code(c.Path), cell(c.Detail))
	}
	if folded {
		b.WriteString("\n</details>\n")
	}
	b.WriteString("\n")
}

func code(s string) string {
	if s == "" {
		return ""
	}
	return "`" + cell(s) + "`"
}

func cell(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

// latestCmd prints the highest non-prerelease version of a provider on a
// registry implementing the provider registry protocol.
func latestCmd(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("latest", flag.ContinueOnError)
	registry := fs.String("registry", "https://registry.terraform.io", "registry base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 || strings.Count(fs.Arg(0), "/") != 1 {
		return errors.New("usage: schema-diff latest [-registry URL] <namespace>/<name>")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	url := strings.TrimSuffix(*registry, "/") + "/v1/providers/" + fs.Arg(0) + "/versions"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	v, err := latestVersion(body)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, v)
	return err
}

func latestVersion(body []byte) (string, error) {
	var doc struct {
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", err
	}
	best := ""
	for _, v := range doc.Versions {
		sv := "v" + v.Version
		if !semver.IsValid(sv) || semver.Prerelease(sv) != "" {
			continue
		}
		if best == "" || semver.Compare(sv, "v"+best) > 0 {
			best = v.Version
		}
	}
	if best == "" {
		return "", errors.New("no released version found")
	}
	return best, nil
}
