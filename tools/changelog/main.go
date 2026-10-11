// Command changelog maintains CHANGELOG.md (Keep a Changelog format) for the
// release workflow.
//
//	changelog prepare -version 0.2.0 [-date 2026-10-01] [-entries FILE] [-file CHANGELOG.md]
//
// turns the [Unreleased] section into a "## [0.2.0] - <date>" section, merging
// in the entries drafted by git-cliff (-entries, "### Group" blocks; "-" reads
// stdin). Hand-written [Unreleased] entries come first within each group, the
// groups follow the Keep a Changelog order, and the link references at the
// bottom of the file gain the new version.
//
//	changelog notes -version 0.2.0 [-file CHANGELOG.md]
//
// prints the body of one version's section, which the release workflow passes
// to GoReleaser as the GitHub Release notes. It fails when the section is
// missing or empty, so a tag cannot be released without a changelog entry.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
)

// groupOrder is the order of the "###" groups in a release section; groups not
// listed here follow in the order they first appear.
var groupOrder = []string{"Breaking changes", "Added", "Changed", "Deprecated", "Removed", "Fixed", "Security"}

var (
	semverRe     = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)
	unreleasedRe = regexp.MustCompile(`^## \[Unreleased\]\s*$`)
	versionRe    = regexp.MustCompile(`^## \[([^\]]+)\]`)
	linkRefRe    = regexp.MustCompile(`^\[[^\]]+\]: \S`)
	// [Unreleased]: https://github.com/<owner>/<repo>/compare/v0.1.1...HEAD
	unreleasedLinkRe = regexp.MustCompile(`^\[Unreleased\]: (\S+)/compare/(\S+)\.\.\.HEAD\s*$`)
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "changelog:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: changelog prepare|notes -version X.Y.Z [flags]")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	file := fs.String("file", "CHANGELOG.md", "changelog to read (and, for prepare, rewrite)")
	version := fs.String("version", "", "release version, with or without a leading v")
	date := fs.String("date", time.Now().UTC().Format(time.DateOnly), "release date (prepare)")
	entries := fs.String("entries", "", `git-cliff "### Group" blocks to merge in, "-" for stdin (prepare)`)
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	v := strings.TrimPrefix(*version, "v")
	if !semverRe.MatchString(v) {
		return fmt.Errorf("-version %q is not a semantic version", *version)
	}
	src, err := os.ReadFile(*file) //nolint:gosec // path comes from the -file flag
	if err != nil {
		return err
	}

	switch args[0] {
	case "notes":
		body, err := notes(string(src), v)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, body)
		return err
	case "prepare":
		var drafted []byte
		switch *entries {
		case "":
		case "-":
			drafted, err = io.ReadAll(stdin)
		default:
			drafted, err = os.ReadFile(*entries) //nolint:gosec // path comes from the -entries flag
		}
		if err != nil {
			return err
		}
		out, err := prepare(string(src), v, *date, string(drafted))
		if err != nil {
			return err
		}
		return os.WriteFile(*file, []byte(out), 0o644) //nolint:gosec // CHANGELOG.md is a public repository file
	default:
		return fmt.Errorf("unknown command %q (want prepare or notes)", args[0])
	}
}

// notes returns the trimmed body of the "## [version]" section.
func notes(src, version string) (string, error) {
	lines := strings.Split(src, "\n")
	start := -1
	for i, l := range lines {
		if m := versionRe.FindStringSubmatch(l); m != nil && m[1] == version {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return "", fmt.Errorf("no \"## [%s]\" section in the changelog; run the Prepare release workflow (or make changelog) first", version)
	}
	body := strings.TrimSpace(strings.Join(lines[start:sectionEnd(lines, start)], "\n"))
	if body == "" {
		return "", fmt.Errorf("the \"## [%s]\" section is empty", version)
	}
	return body, nil
}

// sectionEnd returns the index of the first line at or after start that begins
// the next "## " section or the link reference block.
func sectionEnd(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") || linkRefRe.MatchString(lines[i]) {
			return i
		}
	}
	return len(lines)
}

func prepare(src, version, date, drafted string) (string, error) {
	lines := strings.Split(src, "\n")
	u := slices.IndexFunc(lines, unreleasedRe.MatchString)
	if u < 0 {
		return "", errors.New(`no "## [Unreleased]" heading in the changelog`)
	}
	for _, l := range lines {
		if m := versionRe.FindStringSubmatch(l); m != nil && m[1] == version {
			return "", fmt.Errorf("the changelog already has a \"## [%s]\" section", version)
		}
	}
	end := sectionEnd(lines, u+1)

	preamble, groups, order := parseGroups(lines[u+1 : end])
	_, draftedGroups, draftedOrder := parseGroups(strings.Split(drafted, "\n"))
	for _, name := range draftedOrder {
		if _, ok := groups[name]; !ok {
			order = append(order, name)
		}
		groups[name] = append(groups[name], draftedGroups[name]...)
	}
	order = sortGroups(order)

	var b strings.Builder
	if p := strings.TrimSpace(strings.Join(preamble, "\n")); p != "" {
		b.WriteString(p + "\n\n")
	}
	for _, name := range order {
		if len(groups[name]) == 0 {
			continue
		}
		b.WriteString("### " + name + "\n\n")
		for _, e := range groups[name] {
			b.WriteString(e + "\n")
		}
		b.WriteString("\n")
	}
	section := strings.TrimSpace(b.String())
	if section == "" {
		return "", errors.New("nothing to release: [Unreleased] is empty and no commit since the last tag is listed by cliff.toml")
	}

	out := slices.Clone(lines[:u+1])
	out = append(out, "", fmt.Sprintf("## [%s] - %s", version, date), "")
	out = append(out, strings.Split(section, "\n")...)
	out = append(out, "")
	out = append(out, updateLinks(lines[end:], version)...)
	return strings.Join(out, "\n"), nil
}

// parseGroups splits a section body into the text before the first "###"
// heading and the entries of each group. An entry is a "- " line plus its
// indented continuation lines.
func parseGroups(lines []string) (preamble []string, groups map[string][]string, order []string) {
	groups = map[string][]string{}
	current := ""
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, "### "):
			current = strings.TrimSpace(strings.TrimPrefix(l, "### "))
			if _, ok := groups[current]; !ok {
				groups[current] = nil
				order = append(order, current)
			}
		case current == "":
			preamble = append(preamble, l)
		case strings.TrimSpace(l) == "":
		case strings.HasPrefix(l, "- ") || strings.HasPrefix(l, "* "):
			groups[current] = append(groups[current], l)
		case len(groups[current]) > 0:
			// Continuation of a wrapped entry.
			groups[current][len(groups[current])-1] += "\n" + l
		default:
			groups[current] = append(groups[current], l)
		}
	}
	return preamble, groups, order
}

// sortGroups orders known groups as in groupOrder and keeps unknown groups, in
// their original order, after them.
func sortGroups(names []string) []string {
	rank := func(n string) int {
		if i := slices.Index(groupOrder, n); i >= 0 {
			return i
		}
		return len(groupOrder)
	}
	out := slices.Clone(names)
	slices.SortStableFunc(out, func(a, b string) int { return rank(a) - rank(b) })
	return out
}

// updateLinks points [Unreleased] at the new tag and adds a compare link for the
// new version. Without an [Unreleased] compare link the lines are returned as is.
func updateLinks(lines []string, version string) []string {
	out := make([]string, 0, len(lines)+1)
	for _, l := range lines {
		m := unreleasedLinkRe.FindStringSubmatch(l)
		if m == nil {
			out = append(out, l)
			continue
		}
		base, prev := m[1], m[2]
		out = append(out,
			fmt.Sprintf("[Unreleased]: %s/compare/v%s...HEAD", base, version),
			fmt.Sprintf("[%s]: %s/compare/%s...v%s", version, base, prev, version),
		)
	}
	return out
}
