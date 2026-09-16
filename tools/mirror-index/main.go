// Command mirror-index turns a GoReleaser dist/ directory into a Terraform
// provider network mirror (https://developer.hashicorp.com/terraform/internals/provider-network-mirror-protocol).
//
// It reads the provider zips from -dist, computes the h1: (dirhash) and zh:
// (zip sha256) hashes Terraform records in .terraform.lock.hcl, and writes
//
//	<out>/<hostname>/<namespace>/<name>/index.json
//	<out>/<hostname>/<namespace>/<name>/<version>.json
//
// When -base-url is empty the zips are copied next to the JSON files and
// referenced with relative URLs, which gives a self-contained directory that
// can be served by any static web server. With -base-url the JSON points at
// the zips hosted elsewhere (typically the GitHub Release assets) and nothing
// is copied.
//
// An existing index.json in the output directory, or one fetched from
// -merge-from (a directory or an https URL of a previously published mirror),
// is merged so that earlier versions stay installable.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/mod/semver"
	"golang.org/x/mod/sumdb/dirhash"
)

// indexJSON is the body of index.json: every version the mirror knows about.
type indexJSON struct {
	Versions map[string]struct{} `json:"versions"`
}

// versionJSON is the body of <version>.json.
type versionJSON struct {
	Archives map[string]archiveJSON `json:"archives"`
}

type archiveJSON struct {
	URL    string   `json:"url"`
	Hashes []string `json:"hashes"`
}

// archive is one provider zip found in dist/.
type archive struct {
	Path     string
	FileName string
	Version  string
	Platform string // os_arch
}

var zipNameRE = regexp.MustCompile(`^(terraform-provider-[a-z0-9-]+)_([^_]+)_([a-z0-9]+_[a-z0-9]+)\.zip$`)

func main() {
	log.SetFlags(0)
	log.SetPrefix("mirror-index: ")

	var (
		dist      = flag.String("dist", "dist", "GoReleaser output directory containing the provider zips")
		out       = flag.String("out", "dist/mirror", "directory the mirror tree is written to")
		baseURL   = flag.String("base-url", "", "absolute URL prefix for the zips; empty copies the zips into the mirror and uses relative URLs")
		hostname  = flag.String("hostname", "registry.terraform.io", "provider source hostname")
		namespace = flag.String("namespace", "elliot", "provider source namespace")
		name      = flag.String("name", "netbox", "provider type name")
		mergeFrom = flag.String("merge-from", "", "directory or https URL of an existing mirror whose versions are carried over")
		mirrorURL = flag.String("mirror-url", "https://example.com/", "public URL of the mirror root, shown in the generated index.html")
		noHTML    = flag.Bool("no-html", false, "do not write a human-readable index.html at the mirror root")
	)
	flag.Parse()

	if err := run(*dist, *out, *baseURL, *hostname, *namespace, *name, *mergeFrom, *mirrorURL, !*noHTML); err != nil {
		log.Fatal(err)
	}
}

func run(dist, out, baseURL, hostname, namespace, name, mergeFrom, mirrorURL string, writeHTML bool) error {
	archives, err := findArchives(dist, "terraform-provider-"+name)
	if err != nil {
		return err
	}
	if len(archives) == 0 {
		return fmt.Errorf("no terraform-provider-%s_*_<os>_<arch>.zip files in %s", name, dist)
	}
	sums, err := readSums(dist)
	if err != nil {
		return err
	}

	providerDir := filepath.Join(out, hostname, namespace, name)
	if err := os.MkdirAll(providerDir, 0o755); err != nil { //nolint:gosec // mirror is world-readable by design
		return err
	}

	index := indexJSON{Versions: map[string]struct{}{}}
	if mergeFrom != "" {
		if err := mergeExisting(mergeFrom, path.Join(hostname, namespace, name), providerDir, &index); err != nil {
			return err
		}
	}
	if err := mergeExisting(out, path.Join(hostname, namespace, name), providerDir, &index); err != nil {
		return err
	}

	byVersion := map[string]*versionJSON{}
	for _, a := range archives {
		v := byVersion[a.Version]
		if v == nil {
			v = &versionJSON{Archives: map[string]archiveJSON{}}
			byVersion[a.Version] = v
		}
		h1, err := dirhash.HashZip(a.Path, dirhash.Hash1)
		if err != nil {
			return fmt.Errorf("%s: %w", a.FileName, err)
		}
		hashes := []string{h1}
		zh, err := zipSHA256(a.Path)
		if err != nil {
			return fmt.Errorf("%s: %w", a.FileName, err)
		}
		if want, ok := sums[a.FileName]; ok && want != zh {
			return fmt.Errorf("%s: sha256 %s does not match SHA256SUMS entry %s", a.FileName, zh, want)
		}
		hashes = append(hashes, "zh:"+zh)

		u := a.FileName
		if baseURL != "" {
			u = strings.TrimSuffix(baseURL, "/") + "/" + a.FileName
		} else if err := copyFile(a.Path, filepath.Join(providerDir, a.FileName)); err != nil {
			return err
		}
		v.Archives[a.Platform] = archiveJSON{URL: u, Hashes: hashes}
		index.Versions[a.Version] = struct{}{}
	}

	for version, v := range byVersion {
		if err := writeJSON(filepath.Join(providerDir, version+".json"), v); err != nil {
			return err
		}
		platforms := make([]string, 0, len(v.Archives))
		for p := range v.Archives {
			platforms = append(platforms, p)
		}
		sort.Strings(platforms)
		log.Printf("%s/%s/%s %s: %s", hostname, namespace, name, version, strings.Join(platforms, " "))
	}
	if err := writeJSON(filepath.Join(providerDir, "index.json"), index); err != nil {
		return err
	}
	if writeHTML {
		if err := writeIndexHTML(out, hostname, namespace, name, mirrorURL, index); err != nil {
			return err
		}
	}
	return nil
}

func findArchives(dist, prefix string) ([]archive, error) {
	entries, err := os.ReadDir(dist)
	if err != nil {
		return nil, err
	}
	var archives []archive
	for _, e := range entries {
		m := zipNameRE.FindStringSubmatch(e.Name())
		if m == nil || m[1] != prefix {
			continue
		}
		archives = append(archives, archive{
			Path:     filepath.Join(dist, e.Name()),
			FileName: e.Name(),
			Version:  m[2],
			Platform: m[3],
		})
	}
	sort.Slice(archives, func(i, j int) bool { return archives[i].FileName < archives[j].FileName })
	return archives, nil
}

// readSums parses every *_SHA256SUMS file in dist into filename -> hex digest.
func readSums(dist string) (map[string]string, error) {
	matches, err := filepath.Glob(filepath.Join(dist, "*_SHA256SUMS"))
	if err != nil {
		return nil, err
	}
	sums := map[string]string{}
	for _, m := range matches {
		f, err := os.Open(m) //nolint:gosec // path comes from the -dist flag
		if err != nil {
			return nil, err
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			fields := strings.Fields(sc.Text())
			if len(fields) != 2 {
				continue
			}
			sums[strings.TrimPrefix(fields[1], "*")] = fields[0]
		}
		if err := errors.Join(sc.Err(), f.Close()); err != nil {
			return nil, err
		}
	}
	return sums, nil
}

func zipSHA256(p string) (string, error) {
	f, err := os.Open(p) //nolint:gosec // path comes from the -dist flag
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck // read-only
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// mergeExisting loads <from>/<rel>/index.json and, for versions that are not
// produced by this run, copies their <version>.json (and relative zips) into
// providerDir. `from` is a directory or an http(s) URL. A missing index is not
// an error: the first publish has nothing to merge.
func mergeExisting(from, rel, providerDir string, index *indexJSON) error {
	fetch := fileFetcher(from, rel)
	if strings.HasPrefix(from, "http://") || strings.HasPrefix(from, "https://") {
		fetch = httpFetcher(from, rel)
	}
	data, err := fetch("index.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("merge %s: %w", from, err)
	}
	var existing indexJSON
	if err := json.Unmarshal(data, &existing); err != nil {
		return fmt.Errorf("merge %s: index.json: %w", from, err)
	}
	for version := range existing.Versions {
		if _, done := index.Versions[version]; done {
			continue
		}
		target := filepath.Join(providerDir, version+".json")
		if _, err := os.Stat(target); err == nil {
			index.Versions[version] = struct{}{}
			continue
		}
		body, err := fetch(version + ".json")
		if err != nil {
			log.Printf("warning: dropping %s from the index: %v", version, err)
			continue
		}
		var v versionJSON
		if err := json.Unmarshal(body, &v); err != nil {
			log.Printf("warning: dropping %s from the index: %v", version, err)
			continue
		}
		// Relative archive URLs must travel with the JSON.
		for platform, a := range v.Archives {
			if u, err := url.Parse(a.URL); err != nil || u.IsAbs() {
				continue
			}
			zipData, err := fetch(a.URL)
			if err != nil {
				log.Printf("warning: %s %s: %v", version, platform, err)
				continue
			}
			if err := os.WriteFile(filepath.Join(providerDir, path.Base(a.URL)), zipData, 0o644); err != nil { //nolint:gosec // mirror content
				return err
			}
		}
		if err := writeJSON(target, v); err != nil {
			return err
		}
		index.Versions[version] = struct{}{}
		log.Printf("carried over %s from %s", version, from)
	}
	return nil
}

func fileFetcher(dir, rel string) func(string) ([]byte, error) {
	return func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel), name)) //nolint:gosec // operator-supplied path
	}
}

func httpFetcher(base, rel string) func(string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	return func(name string) ([]byte, error) {
		u := strings.TrimSuffix(base, "/") + "/" + rel + "/" + name
		resp, err := client.Get(u) //nolint:noctx // short-lived CLI
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close() //nolint:errcheck // read-only
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%s: %w", u, os.ErrNotExist)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%s: HTTP %s", u, resp.Status)
		}
		return io.ReadAll(resp.Body)
	}
}

func writeJSON(p string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644) //nolint:gosec // mirror content is public
}

func copyFile(src, dst string) error {
	in, err := os.Open(src) //nolint:gosec // path comes from the -dist flag
	if err != nil {
		return err
	}
	defer in.Close()                                                        //nolint:errcheck // read-only
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644) //nolint:gosec // mirror content is public
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

var indexHTML = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>{{.Source}} provider mirror</title>
<style>body{font:15px/1.5 system-ui,sans-serif;max-width:52rem;margin:3rem auto;padding:0 1rem}pre{background:#f4f4f4;padding:.75rem;overflow:auto}</style>
</head><body>
<h1>Terraform provider network mirror</h1>
<p>This site serves <code>{{.Source}}</code> using the
<a href="https://developer.hashicorp.com/terraform/internals/provider-network-mirror-protocol">provider network mirror protocol</a>.</p>
<h2>Use it</h2>
<p>Add to <code>~/.terraformrc</code> (or a file named by <code>TF_CLI_CONFIG_FILE</code>):</p>
<pre>provider_installation {
  network_mirror {
    url     = "{{.URL}}"
    include = ["{{.Source}}"]
  }
  direct {
    exclude = ["{{.Source}}"]
  }
}</pre>
<p>Then reference the provider as usual:</p>
<pre>terraform {
  required_providers {
    {{.Name}} = {
      source  = "{{.Namespace}}/{{.Name}}"
      version = "~&gt; {{.Latest}}"
    }
  }
}</pre>
<h2>Versions</h2>
<ul>{{range .Versions}}<li><a href="{{$.Rel}}/{{.}}.json">{{.}}</a></li>{{end}}</ul>
<p><a href="{{.Rel}}/index.json">index.json</a></p>
</body></html>
`))

func writeIndexHTML(out, hostname, namespace, name, mirrorURL string, index indexJSON) error {
	versions := make([]string, 0, len(index.Versions))
	for v := range index.Versions {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool { return semver.Compare("v"+versions[i], "v"+versions[j]) > 0 })
	latest := ""
	if len(versions) > 0 {
		latest = versions[0]
	}
	f, err := os.Create(filepath.Join(out, "index.html")) //nolint:gosec // output dir from flag
	if err != nil {
		return err
	}
	data := map[string]any{
		"Source":    namespace + "/" + name,
		"Namespace": namespace,
		"Name":      name,
		"Rel":       path.Join(hostname, namespace, name),
		"URL":       mirrorURL,
		"Latest":    latest,
		"Versions":  versions,
	}
	if err := indexHTML.Execute(f, data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
