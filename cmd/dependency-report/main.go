package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"
)

const cooldownDays = 14

// Module represents the JSON output of `go list -m -u -json`.
type Module struct {
	Path     string  `json:"Path"`
	Version  string  `json:"Version"`
	Indirect bool    `json:"Indirect"`
	Update   *Update `json:"Update"`
}

type Update struct {
	Version string `json:"Version"`
}

type outdatedModule struct {
	Path    string
	Current string
	Update  string // best upgrade target (may differ from go list's latest)
	Latest  string // absolute latest from go list
}

// proxyInfo is the response from the Go module proxy /{module}/@v/{version}.info
type proxyInfo struct {
	Version string    `json:"Version"`
	Time    time.Time `json:"Time"`
}

func main() {
	outputMode := flag.String("o", "markdown", "output mode: markdown (default) or list")
	flag.Parse()

	if err := run(*outputMode); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(outputMode string) error {
	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -cooldownDays)

	modules, err := listOutdated()
	if err != nil {
		return fmt.Errorf("list outdated modules: %w", err)
	}

	resolved := resolveUpgrades(modules, cutoff)

	switch outputMode {
	case "list":
		return writeList(os.Stdout, resolved)
	case "markdown":
		return writeMarkdown(resolved)
	default:
		return fmt.Errorf("unknown output mode: %s (use 'markdown' or 'list')", outputMode)
	}
}

// writeList prints space-separated module@version pairs suitable for use with go get.
func writeList(w io.Writer, resolved []resolvedModule) error {
	if len(resolved) == 0 {
		return nil
	}
	args := make([]string, len(resolved))
	for i, m := range resolved {
		args[i] = m.Path + "@" + m.Update
	}
	_, err := fmt.Fprintln(w, strings.Join(args, " "))
	return err
}

// writeMarkdown writes a GitHub-flavored Markdown report to GITHUB_STEP_SUMMARY or stdout.
func writeMarkdown(resolved []resolvedModule) error {
	var w io.Writer = os.Stdout
	if summaryFile := os.Getenv("GITHUB_STEP_SUMMARY"); summaryFile != "" {
		f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open step summary: %w", err)
		}
		defer func() { _ = f.Close() }()
		w = f
	}

	_, _ = fmt.Fprintln(w, "## Outdated Go Dependencies")
	_, _ = fmt.Fprintln(w)

	if len(resolved) == 0 {
		_, _ = fmt.Fprintf(w, "No direct dependencies have updates past the %d-day cooldown.\n", cooldownDays)
	} else {
		_, _ = fmt.Fprintln(w, "| Module | Current | Upgrade to | Latest | Published | Upgrade command |")
		_, _ = fmt.Fprintln(w, "|--------|---------|------------|--------|-----------|-----------------|")
		for _, m := range resolved {
			latest := ""
			if m.Update != m.Latest {
				latest = m.Latest
			}
			_, _ = fmt.Fprintf(w, "| `%s` | %s | %s | %s | %s | `go get %s@%s` |\n",
				m.Path, m.Current, m.Update, latest, m.Published, m.Path, m.Update)
		}
	}

	_, _ = fmt.Fprintln(w)
	return nil
}

type resolvedModule struct {
	Path      string
	Current   string
	Update    string // version to upgrade to (past cooldown)
	Latest    string // absolute latest from go list
	Published string // publish date of Update
}

// resolveUpgrades finds the best upgrade target for each module, respecting cooldown.
func resolveUpgrades(modules []outdatedModule, cutoff time.Time) []resolvedModule {
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	var results []resolvedModule

	for _, m := range modules {
		wg.Add(1)
		go func(m outdatedModule) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			r, ok := findUpgrade(m, cutoff)
			if !ok {
				return
			}
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(m)
	}
	wg.Wait()

	return results
}

// findUpgrade checks the latest version first; if it's past cooldown, use it.
// Otherwise, fetches the version list and walks backwards to find the newest
// version newer than current that is past cooldown.
func findUpgrade(m outdatedModule, cutoff time.Time) (resolvedModule, bool) {
	// Try the latest version first (avoids extra proxy calls in the common case)
	if t, err := queryProxyInfo(m.Path, m.Update); err == nil && !t.After(cutoff) {
		return resolvedModule{
			Path:      m.Path,
			Current:   m.Current,
			Update:    m.Update,
			Latest:    m.Update,
			Published: t.Format("2006-01-02"),
		}, true
	}

	// Latest is too fresh — walk backwards through the version list
	versions, err := queryProxyVersions(m.Path)
	if err != nil {
		return resolvedModule{}, false
	}

	// Versions from the proxy are in chronological order; walk from newest to oldest
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		if !semver.IsValid(v) {
			continue
		}
		// Skip pre-release versions
		if semver.Prerelease(v) != "" {
			continue
		}
		// Must be newer than current
		if semver.Compare(v, m.Current) <= 0 {
			break // all remaining versions are older; stop
		}
		t, err := queryProxyInfo(m.Path, v)
		if err != nil {
			continue
		}
		if !t.After(cutoff) {
			return resolvedModule{
				Path:      m.Path,
				Current:   m.Current,
				Update:    v,
				Latest:    m.Update,
				Published: t.Format("2006-01-02"),
			}, true
		}
	}

	return resolvedModule{}, false
}

// listOutdated runs `go list -m -u -json all` and parses the streaming JSON output.
func listOutdated() ([]outdatedModule, error) {
	cmd := exec.Command("go", "list", "-m", "-u", "-json", "all")
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var result []outdatedModule
	dec := json.NewDecoder(stdout)
	for dec.More() {
		var m Module
		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("decode module: %w", err)
		}
		if m.Update == nil || m.Indirect {
			continue
		}
		result = append(result, outdatedModule{
			Path:    m.Path,
			Current: m.Version,
			Update:  m.Update.Version,
			Latest:  m.Update.Version,
		})
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}
	return result, nil
}

// queryProxyInfo fetches the publish time of a module version from the Go module proxy.
func queryProxyInfo(modulePath, version string) (time.Time, error) {
	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.info", modulePath, version)
	resp, err := http.Get(url)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("proxy returned %d", resp.StatusCode)
	}

	var info proxyInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return time.Time{}, err
	}
	return info.Time, nil
}

// queryProxyVersions fetches the list of known versions for a module from the Go module proxy.
func queryProxyVersions(modulePath string) ([]string, error) {
	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/list", modulePath)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxy returned %d", resp.StatusCode)
	}

	var versions []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if v := scanner.Text(); v != "" {
			versions = append(versions, v)
		}
	}
	return versions, scanner.Err()
}
