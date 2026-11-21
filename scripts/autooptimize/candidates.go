package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Candidate kinds, in the order they are picked: lint findings first because
// they are the most mechanical, coverage gaps last because they add code.
const (
	KindLint     = "lint"
	KindDeadcode = "deadcode"
	KindDocs     = "docs"
	KindCoverage = "coverage"
)

var kindRank = map[string]int{KindLint: 0, KindDeadcode: 1, KindDocs: 2, KindCoverage: 3}

// docFiles are the only files a docs candidate may change.
var docFiles = []string{"README.md", "README_zh.md"}

// Candidate is one deterministic finding the agent may work on.
type Candidate struct {
	ID       string  `json:"id"`
	Kind     string  `json:"kind"`
	Package  string  `json:"package,omitempty"` // repo-relative package directory; empty for docs
	File     string  `json:"file,omitempty"`
	Line     int     `json:"line,omitempty"`
	Source   string  `json:"source"`
	Message  string  `json:"message"`
	Coverage float64 `json:"coverage,omitempty"`
}

// Title is a one-line summary used for the branch, commit and PR title.
func (c Candidate) Title() string {
	switch c.Kind {
	case KindLint:
		return fmt.Sprintf("fix %s finding in %s", c.Source, c.Package)
	case KindDeadcode:
		return fmt.Sprintf("remove unreachable code in %s", c.Package)
	case KindDocs:
		return "fix README flag drift"
	case KindCoverage:
		return fmt.Sprintf("add tests for %s", c.Package)
	}
	return c.Message
}

// AllowedPaths describes, for people and for the agent prompt, which files the
// change may touch. checkDiff enforces the same rule.
func (c Candidate) AllowedPaths() string {
	switch c.Kind {
	case KindDocs:
		return strings.Join(docFiles, ", ")
	case KindCoverage:
		return fmt.Sprintf("*_test.go files directly in %s/ and files under %s/testdata/", c.Package, c.Package)
	default:
		return fmt.Sprintf(".go files directly in %s/ and files under %s/testdata/", c.Package, c.Package)
	}
}

// excludedPrefixes are never candidates and never allowed in a diff: the
// automation itself, CI config, release packaging, and planning docs.
var excludedPrefixes = []string{
	".github/", ".claude/", ".codex/", ".trae/", "scripts/", "npm/", "docs/",
	"workspace/", "openspec/", "skills/", "vendor/",
}

// excludedFiles are top-level files a change must never touch.
var excludedFiles = map[string]bool{
	"go.mod": true, "go.sum": true, "Makefile": true, "AGENTS.md": true,
	"CLAUDE.md": true, ".golangci.yml": true, ".goreleaser.yml": true, ".gitignore": true,
}

func isExcludedPath(p string) bool {
	if excludedFiles[p] {
		return true
	}
	for _, pre := range excludedPrefixes {
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

// cleanPath turns a tool-reported path into a clean, slash-separated,
// repo-relative path ("./cmd/a.go" -> "cmd/a.go"); empty stays empty.
func cleanPath(p string) string {
	if p == "" {
		return ""
	}
	return path.Clean(filepath.ToSlash(p))
}

func pkgDir(file string) string {
	d := path.Dir(filepath.ToSlash(file))
	return d
}

func candidateID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])[:12]
}

func finalize(c Candidate) Candidate {
	// The line number is left out of the ID so unrelated edits above a finding
	// do not turn it into a "new" candidate. A coverage candidate is the
	// package itself: its percentage changes with every edit, so it is left
	// out too, and a rejected coverage PR stays rejected.
	msg := c.Message
	if c.Kind == KindCoverage {
		msg = ""
	}
	c.ID = candidateID(c.Kind, c.Package, c.File, c.Source, msg)
	return c
}

// sortCandidates orders candidates by kind, then by the most useful order
// within a kind, then by a stable tie-breaker.
func sortCandidates(cs []Candidate) {
	sort.SliceStable(cs, func(i, j int) bool {
		a, b := cs[i], cs[j]
		if kindRank[a.Kind] != kindRank[b.Kind] {
			return kindRank[a.Kind] < kindRank[b.Kind]
		}
		if a.Kind == KindCoverage && a.Coverage != b.Coverage {
			return a.Coverage < b.Coverage
		}
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.ID < b.ID
	})
}

// dedupe keeps the first candidate for each ID.
func dedupe(cs []Candidate) []Candidate {
	seen := map[string]bool{}
	out := cs[:0]
	for _, c := range cs {
		if seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		out = append(out, c)
	}
	return out
}

// parseLint reads golangci-lint v2 JSON output (--output.json.path).
func parseLint(data []byte) ([]Candidate, error) {
	var report struct {
		Issues []struct {
			FromLinter string
			Text       string
			Pos        struct {
				Filename string
				Line     int
			}
		}
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse golangci-lint json: %w", err)
	}
	var out []Candidate
	for _, is := range report.Issues {
		file := cleanPath(is.Pos.Filename)
		if file == "" || isExcludedPath(file) || strings.HasPrefix(file, "testdata/") || strings.Contains(file, "/testdata/") {
			continue
		}
		out = append(out, finalize(Candidate{
			Kind: KindLint, Package: pkgDir(file), File: file, Line: is.Pos.Line,
			Source: "golangci-lint/" + is.FromLinter, Message: strings.TrimSpace(is.Text),
		}))
	}
	return out, nil
}

// parseDeadcode reads `deadcode -json ./...` output. Run without -test it
// reports functions only tests reach, which is production-dead code.
func parseDeadcode(data []byte) ([]Candidate, error) {
	var pkgs []struct {
		Path  string
		Funcs []struct {
			Name      string
			Generated bool
			Position  struct {
				File string
				Line int
			}
		}
	}
	if err := json.Unmarshal(data, &pkgs); err != nil {
		return nil, fmt.Errorf("parse deadcode json: %w", err)
	}
	var out []Candidate
	for _, p := range pkgs {
		// Test helper packages exist to be used only by tests.
		if strings.HasSuffix(p.Path, "/testutil") {
			continue
		}
		for _, f := range p.Funcs {
			file := cleanPath(f.Position.File)
			if f.Generated || file == "" || isExcludedPath(file) || strings.HasSuffix(file, "_test.go") {
				continue
			}
			out = append(out, finalize(Candidate{
				Kind: KindDeadcode, Package: pkgDir(file), File: file, Line: f.Position.Line,
				Source:  "deadcode",
				Message: fmt.Sprintf("%s is unreachable from main (only tests or nothing call it)", f.Name),
			}))
		}
	}
	return out, nil
}

var coverLine = regexp.MustCompile(`^ok\s+(\S+)\s+\S+\s+coverage:\s+([0-9.]+)% of statements`)

// parseCoverage reads `go test -cover` output and reports packages that have
// tests but fall below minPct. Packages without any test file are skipped:
// starting a first test suite is a design decision, not a cleanup.
func parseCoverage(data []byte, module string, minPct float64) ([]Candidate, error) {
	var out []Candidate
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		m := coverLine.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		pct, err := strconv.ParseFloat(m[2], 64)
		if err != nil || pct >= minPct {
			continue
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(m[1], module), "/")
		if rel == "" || rel == m[1] {
			continue // root main package or a foreign module
		}
		if isExcludedPath(rel + "/") {
			continue
		}
		out = append(out, finalize(Candidate{
			Kind: KindCoverage, Package: rel, Source: "go test -cover", Coverage: pct,
			Message: fmt.Sprintf("statement coverage %.1f%% is below %.0f%%", pct, minPct),
		}))
	}
	return out, sc.Err()
}

var (
	// Matches cobra/pflag definitions such as Flags().String("name", ...),
	// PersistentFlags().BoolVarP(&v, "name", ...). Lookups (GetString, Set,
	// Changed) are not definitions and do not match.
	flagDef = regexp.MustCompile(`(?:Persistent)?Flags\(\)\.(?:String|Bool|Int|Int64|Float64|Duration|StringSlice|StringArray|IntSlice|Count)(?:Var)?P?\(\s*(?:&?[\w.]+\s*,\s*)?"([a-z0-9][a-z0-9-]*)"`)
	docFlag = regexp.MustCompile(`(?:^|[^\w-])--([a-z][a-z0-9-]*[a-z0-9])`)
)

// ignoredDocFlags are flags a README may mention that cmd/ never defines:
// cobra built-ins and the placeholder in "Adding a new provider".
var ignoredDocFlags = map[string]bool{"help": true, "version": true, "myprovider-dir": true}

// collectDocDrift compares the long flags mentioned in the READMEs with the
// flags defined by non-test files in cmd/.
func collectDocDrift(repo string) ([]Candidate, error) {
	defined := map[string]bool{}
	goFiles, err := filepath.Glob(filepath.Join(repo, "cmd", "*.go"))
	if err != nil {
		return nil, err
	}
	for _, f := range goFiles {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		for _, m := range flagDef.FindAllSubmatch(src, -1) {
			defined[string(m[1])] = true
		}
	}
	if len(defined) == 0 {
		return nil, fmt.Errorf("no flag definitions found under %s/cmd", repo)
	}

	mentioned := map[string]map[string]bool{} // flag -> docs that mention it
	for _, doc := range docFiles {
		src, err := os.ReadFile(filepath.Join(repo, doc))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, m := range docFlag.FindAllSubmatch(src, -1) {
			name := string(m[1])
			if mentioned[name] == nil {
				mentioned[name] = map[string]bool{}
			}
			mentioned[name][doc] = true
		}
	}

	var out []Candidate
	for name, docs := range mentioned {
		if defined[name] || ignoredDocFlags[name] {
			continue
		}
		for _, doc := range sortedKeys(docs) {
			out = append(out, finalize(Candidate{
				Kind: KindDocs, File: doc, Source: "doc-drift",
				Message: fmt.Sprintf("%s mentions --%s, but no command in cmd/ defines it", doc, name),
			}))
		}
	}
	for name := range defined {
		for _, doc := range docFiles {
			if mentioned[name][doc] {
				continue
			}
			if _, err := os.Stat(filepath.Join(repo, doc)); err != nil {
				continue
			}
			out = append(out, finalize(Candidate{
				Kind: KindDocs, File: doc, Source: "doc-drift",
				Message: fmt.Sprintf("cmd/ defines --%s, but %s never mentions it", name, doc),
			}))
		}
	}
	return out, nil
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// readModule returns the module path declared in go.mod.
func readModule(repo string) (string, error) {
	data, err := os.ReadFile(filepath.Join(repo, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("no module line in go.mod")
}
