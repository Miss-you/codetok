package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const module = "github.com/miss-you/codetok"

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func messages(cs []Candidate) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Kind + " " + c.Package + " " + c.File + " " + c.Message
	}
	return out
}

func TestParseLintSkipsExcludedAndTestdataPaths(t *testing.T) {
	cs, err := parseLint(mustRead(t, "lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 2 {
		t.Fatalf("got %d candidates, want 2: %v", len(cs), messages(cs))
	}
	if cs[0].Package != "provider/kimi" || cs[0].Source != "golangci-lint/errcheck" || cs[0].Line != 40 {
		t.Errorf("first candidate = %+v", cs[0])
	}
	dotted, err := parseLint([]byte(`{"Issues":[{"FromLinter":"x","Text":"t","Pos":{"Filename":"./scripts/a.go","Line":1}},{"FromLinter":"x","Text":"t","Pos":{"Filename":"./cmd/b.go","Line":1}}]}`))
	if err != nil || len(dotted) != 1 || dotted[0].File != "cmd/b.go" || dotted[0].Package != "cmd" {
		t.Errorf("./-prefixed paths: %+v %v", dotted, err)
	}
	if _, err := parseLint([]byte("not json")); err == nil {
		t.Error("want an error for malformed JSON")
	}
}

func TestParseDeadcodeSkipsGeneratedAndTestutil(t *testing.T) {
	cs, err := parseDeadcode(mustRead(t, "deadcode.json"))
	if err != nil {
		t.Fatal(err)
	}
	got := messages(cs)
	want := []string{
		"deadcode stats stats/aggregator.go FilterByDateRange is unreachable from main (only tests or nothing call it)",
		"deadcode cmd cmd/collect.go collectUsageEvents is unreachable from main (only tests or nothing call it)",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if cs, err := parseDeadcode([]byte("null")); err != nil || len(cs) != 0 {
		t.Errorf("deadcode null output: %v %v", cs, err)
	}
}

func TestParseCoverageReportsOnlyTestedPackagesBelowThreshold(t *testing.T) {
	cs, err := parseCoverage(mustRead(t, "cover.txt"), module, 75)
	if err != nil {
		t.Fatal(err)
	}
	// cursor (72.3) and provider (55.0) qualify. The root package and
	// internal/testutil have no tests, e2e has no statements, scripts/ is
	// excluded, and stats is above the threshold.
	var pkgs []string
	for _, c := range cs {
		pkgs = append(pkgs, c.Package)
	}
	if !reflect.DeepEqual(pkgs, []string{"cursor", "provider"}) {
		t.Errorf("packages = %v", pkgs)
	}
}

func TestCollectDocDrift(t *testing.T) {
	cs, err := collectDocDrift(filepath.Join("testdata", "repo"))
	if err != nil {
		t.Fatal(err)
	}
	sortCandidates(cs)
	got := map[string]bool{}
	for _, c := range cs {
		got[c.Message] = true
		if c.Kind != KindDocs || c.Package != "" {
			t.Errorf("bad docs candidate %+v", c)
		}
	}
	want := []string{
		"README.md mentions --legacy-mode, but no command in cmd/ defines it",
		"cmd/ defines --token, but README.md never mentions it",
		"cmd/ defines --verbose, but README.md never mentions it",
		"cmd/ defines --verbose, but README_zh.md never mentions it",
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("missing %q", w)
		}
	}
	if len(cs) != len(want) {
		t.Errorf("got %d candidates, want %d: %v", len(cs), len(want), messages(cs))
	}
	for _, c := range cs {
		for _, bad := range []string{"--help", "--myprovider-dir", "--test-only", "--lookup-only", "--bar"} {
			if strings.Contains(c.Message, bad+",") || strings.Contains(c.Message, bad+" ") {
				t.Errorf("unexpected candidate %q", c.Message)
			}
		}
	}
}

func TestCollectDocDriftNeedsFlagDefinitions(t *testing.T) {
	if _, err := collectDocDrift(t.TempDir()); err == nil {
		t.Error("want an error when cmd/ defines no flags (wrong -repo)")
	}
}

func TestCandidateIDIgnoresLineNumbers(t *testing.T) {
	a := finalize(Candidate{Kind: KindLint, Package: "cmd", File: "cmd/a.go", Line: 3, Source: "x", Message: "m"})
	b := finalize(Candidate{Kind: KindLint, Package: "cmd", File: "cmd/a.go", Line: 30, Source: "x", Message: "m"})
	c := finalize(Candidate{Kind: KindLint, Package: "cmd", File: "cmd/a.go", Line: 3, Source: "x", Message: "other"})
	if a.ID != b.ID {
		t.Error("IDs differ only by line number")
	}
	if a.ID == c.ID || len(a.ID) != 12 {
		t.Errorf("IDs %q %q", a.ID, c.ID)
	}
	cov1 := finalize(Candidate{Kind: KindCoverage, Package: "cursor", Source: "go test -cover", Message: "statement coverage 52.0% is below 60%"})
	cov2 := finalize(Candidate{Kind: KindCoverage, Package: "cursor", Source: "go test -cover", Message: "statement coverage 55.4% is below 60%"})
	if cov1.ID != cov2.ID {
		t.Error("coverage candidate ID changes with the percentage")
	}
}

func TestCollectIsSortedByKindAndDeterministic(t *testing.T) {
	args := []string{"collect", "-repo", "testdata/repo", "-lint", "testdata/lint.json",
		"-deadcode", "testdata/deadcode.json", "-cover", "testdata/cover.txt", "-min-coverage", "75"}
	var first, second bytes.Buffer
	if err := run(args, &first); err != nil {
		t.Fatal(err)
	}
	if err := run(args, &second); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Fatal("collect output is not deterministic")
	}
	var cs []Candidate
	if err := json.Unmarshal(first.Bytes(), &cs); err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, c := range cs {
		if len(kinds) == 0 || kinds[len(kinds)-1] != c.Kind {
			kinds = append(kinds, c.Kind)
		}
	}
	if !reflect.DeepEqual(kinds, []string{KindLint, KindDeadcode, KindDocs, KindCoverage}) {
		t.Errorf("kind order = %v", kinds)
	}
	// Coverage candidates are ordered lowest coverage first.
	last := cs[len(cs)-1]
	if last.Package != "cursor" {
		t.Errorf("last candidate = %+v, want cursor (72.3%% after provider 55%%)", last)
	}
}

func TestCollectEmptyListIsAnArray(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "lint.json")
	if err := os.WriteFile(empty, []byte(`{"Issues":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"collect", "-lint", empty, "-docs=false"}, &out); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "[]" {
		t.Errorf("got %q", out.String())
	}
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSelectSkipsExcludedAndRejectedCandidates(t *testing.T) {
	dir := t.TempDir()
	cands := writeFile(t, dir, "c.json", `[
		{"id":"aaaaaaaaaaaa","kind":"lint"},
		{"id":"bbbbbbbbbbbb","kind":"lint"},
		{"id":"cccccccccccc","kind":"lint"},
		{"id":"dddddddddddd","kind":"docs"}]`)
	exclude := writeFile(t, dir, "abandoned.txt", "# abandoned locally\naaaaaaaaaaaa 2026-11-20 tests failed\n\n")
	closed := writeFile(t, dir, "closed.json", `[
		{"number":1,"body":"x\n<!-- auto-optimize-candidate: bbbbbbbbbbbb -->","mergedAt":null},
		{"number":2,"body":"<!-- auto-optimize-candidate: cccccccccccc -->","mergedAt":"2026-11-01T00:00:00Z"}]`)

	var out bytes.Buffer
	if err := run([]string{"select", "-candidates", cands, "-exclude", exclude, "-closed-prs", closed}, &out); err != nil {
		t.Fatal(err)
	}
	// bbbb was closed unmerged (rejected); cccc was merged, so it may come back
	// only if the tools still report it.
	if !strings.Contains(out.String(), `"cccccccccccc"`) {
		t.Errorf("selected %s", out.String())
	}

	all := writeFile(t, dir, "all.txt", "aaaaaaaaaaaa\nbbbbbbbbbbbb\ncccccccccccc\ndddddddddddd\n")
	err := run([]string{"select", "-candidates", cands, "-exclude", all}, &out)
	var skip skipError
	if !errors.As(err, &skip) {
		t.Fatalf("want skip when everything is excluded, got %v", err)
	}
	// A missing exclude file is an empty list, not an error (first run).
	out.Reset()
	if err := run([]string{"select", "-candidates", cands, "-exclude", filepath.Join(dir, "missing")}, &out); err != nil {
		t.Fatal(err)
	}
}

func TestSelectOnEmptyListSkips(t *testing.T) {
	dir := t.TempDir()
	cands := writeFile(t, dir, "c.json", `[]`)
	err := run([]string{"select", "-candidates", cands}, &bytes.Buffer{})
	var skip skipError
	if !errors.As(err, &skip) {
		t.Fatalf("want skip, got %v", err)
	}
}

func TestGateHonorsOpenPRLimit(t *testing.T) {
	dir := t.TempDir()
	none := writeFile(t, dir, "none.json", `[]`)
	one := writeFile(t, dir, "one.json", `[{"number":61,"url":"https://github.com/Miss-you/codetok/pull/61"}]`)
	if err := run([]string{"gate", "-open-prs", none}, &bytes.Buffer{}); err != nil {
		t.Errorf("no open PR should pass: %v", err)
	}
	err := run([]string{"gate", "-open-prs", one}, &bytes.Buffer{})
	var skip skipError
	if !errors.As(err, &skip) || !strings.Contains(skip.reason, "#61") {
		t.Errorf("one open PR should skip with its number, got %v", err)
	}
	if err := run([]string{"gate", "-open-prs", one, "-max-open", "2"}, &bytes.Buffer{}); err != nil {
		t.Errorf("limit 2 with one open PR should pass: %v", err)
	}
}

func TestCheckDiff(t *testing.T) {
	lint := Candidate{Kind: KindLint, Package: "provider/kimi"}
	cover := Candidate{Kind: KindCoverage, Package: "cursor"}
	docs := Candidate{Kind: KindDocs}

	tests := []struct {
		name    string
		c       Candidate
		numstat string
		max     int
		wantBad []string // substrings, one per expected problem
	}{
		{"lint in package", lint, "3\t1\tprovider/kimi/parser.go\n2\t0\tprovider/kimi/parser_test.go\n", 200, nil},
		{"lint testdata", lint, "1\t1\tprovider/kimi/testdata/a.jsonl\n", 200, nil},
		{"lint subpackage", lint, "1\t1\tprovider/kimi/sub/x.go\n", 200, []string{"outside the allowed scope"}},
		{"lint other package", lint, "1\t1\tprovider/parser.go\n", 200, []string{"outside the allowed scope"}},
		{"lint non-go file", lint, "1\t1\tprovider/kimi/notes.md\n", 200, []string{"outside the allowed scope"}},
		{"go.mod", lint, "1\t0\tprovider/kimi/a.go\n1\t1\tgo.mod\n", 200, []string{"never changed"}},
		{"workflow", lint, "1\t1\t.github/workflows/ci.yml\n", 200, []string{"never changed"}},
		{"binary", lint, "-\t-\tprovider/kimi/testdata/blob.bin\n", 200, []string{"binary"}},
		{"too big", lint, "150\t60\tprovider/kimi/parser.go\n", 200, []string{"210 changed lines"}},
		{"at cap", lint, "150\t50\tprovider/kimi/parser.go\n", 200, nil},
		{"empty", lint, "", 200, []string{"empty"}},
		{"coverage test file", cover, "40\t0\tcursor/sync_test.go\n", 200, nil},
		{"coverage prod file", cover, "40\t0\tcursor/sync_test.go\n1\t1\tcursor/sync.go\n", 200, []string{"outside the allowed scope"}},
		{"docs readmes", docs, "2\t1\tREADME.md\n2\t1\tREADME_zh.md\n", 200, nil},
		{"root package testdata", Candidate{Kind: KindLint, Package: "."}, "1\t1\ttestdata/a.json\n1\t1\tmain.go\n", 200, nil},
		{"docs code", docs, "2\t1\tREADME.md\n1\t1\tcmd/daily.go\n", 200, []string{"outside the allowed scope"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := parseNumstat([]byte(tt.numstat))
			if err != nil {
				t.Fatal(err)
			}
			got := checkDiff(tt.c, entries, tt.max)
			if len(got) != len(tt.wantBad) {
				t.Fatalf("problems = %q, want %d", got, len(tt.wantBad))
			}
			for i, want := range tt.wantBad {
				if !strings.Contains(got[i], want) {
					t.Errorf("problem %q does not mention %q", got[i], want)
				}
			}
		})
	}
	if _, err := parseNumstat([]byte("garbage\n")); err == nil {
		t.Error("want an error for malformed numstat")
	}
}

func TestCheckCommandSkipsOnViolation(t *testing.T) {
	dir := t.TempDir()
	cand := writeFile(t, dir, "c.json", `{"id":"aaaaaaaaaaaa","kind":"lint","package":"cmd"}`)
	ok := writeFile(t, dir, "ok.txt", "1\t1\tcmd/daily.go\n")
	if err := run([]string{"check", "-candidate", cand, "-numstat", ok}, &bytes.Buffer{}); err != nil {
		t.Errorf("in-scope diff: %v", err)
	}
	var out bytes.Buffer
	listFile := filepath.Join(dir, "out-of-scope.txt")
	mixed := writeFile(t, dir, "mixed.txt", "1\t1\tcmd/daily.go\n1\t1\tstats/a.go\n2\t0\tgo.mod\n")
	err := run([]string{"check", "-candidate", cand, "-numstat", mixed, "-out-of-scope", listFile}, &out)
	var skip skipError
	if !errors.As(err, &skip) || !strings.Contains(out.String(), "stats/a.go") {
		t.Errorf("out-of-scope diff: err=%v out=%q", err, out.String())
	}
	if data, _ := os.ReadFile(listFile); string(data) != "stats/a.go\ngo.mod\n" {
		t.Errorf("out-of-scope list = %q", data)
	}
	// A size-only violation lists no paths to revert.
	big := writeFile(t, dir, "big.txt", "300\t0\tcmd/daily.go\n")
	_ = run([]string{"check", "-candidate", cand, "-numstat", big, "-out-of-scope", listFile}, &bytes.Buffer{})
	if data, _ := os.ReadFile(listFile); len(data) != 0 {
		t.Errorf("size-only violation listed %q", data)
	}
}

func TestRoundBudget(t *testing.T) {
	state := filepath.Join(t.TempDir(), "rounds")
	for want := 1; want <= 2; want++ {
		var out bytes.Buffer
		if err := run([]string{"round", "-state", state, "-max", "2"}, &out); err != nil {
			t.Fatalf("round %d: %v", want, err)
		}
		if strings.TrimSpace(out.String()) != []string{"", "1", "2"}[want] {
			t.Errorf("round output %q, want %d", out.String(), want)
		}
	}
	err := run([]string{"round", "-state", state, "-max", "2"}, &bytes.Buffer{})
	var skip skipError
	if !errors.As(err, &skip) {
		t.Fatalf("third round should skip, got %v", err)
	}
	// The counter does not move past the budget.
	if data, _ := os.ReadFile(state); strings.TrimSpace(string(data)) != "2" {
		t.Errorf("counter = %q", data)
	}
	bad := writeFile(t, t.TempDir(), "rounds", "x")
	if err := run([]string{"round", "-state", bad}, &bytes.Buffer{}); err == nil || errors.As(err, &skip) {
		t.Errorf("corrupt counter should be an error, got %v", err)
	}
}

func TestReviewVerdict(t *testing.T) {
	tests := []struct {
		file string
		pass bool
		want string
	}{
		{"pass.md", true, "with fixes"},
		{"important.md", false, "2 Critical/Important"},
		{"prose.md", false, "1 Critical/Important"},
		{"no.md", false, "No"},
		{"noverdict.md", false, "no \"Ready to merge?\""},
		{"bold.md", false, "1 Critical/Important"},
		{"hashline.md", false, "Critical/Important"},
		{"noneof.md", false, "1 Critical/Important"},
		{"nosections.md", false, "no Critical and Important sections"},
		{"summarytitle.md", true, "no Critical/Important"},
		{"trace.md", true, "no Critical/Important"},
		{"boldheading.md", false, "1 Critical/Important"},
		{"traceverdict.md", false, "no \"Ready to merge?\""},
		{"fenceverdict.md", false, "reviewer answered \"Ready to merge? No\""},
		{"colonverdict.md", true, "no Critical/Important"},
		{"boldinline.md", false, "1 Critical/Important"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			pass, reason := reviewVerdict(string(mustRead(t, filepath.Join("reviews", tt.file))))
			if pass != tt.pass || !strings.Contains(reason, tt.want) {
				t.Errorf("pass=%v reason=%q", pass, reason)
			}
		})
	}
}

func TestDescribeAndMarker(t *testing.T) {
	dir := t.TempDir()
	cand := writeFile(t, dir, "c.json",
		`{"id":"0123456789ab","kind":"deadcode","package":"stats","file":"stats/a.go","line":7,"source":"deadcode","message":"m"}`)
	for field, want := range map[string]string{
		"id": "0123456789ab", "kind": "deadcode", "package": "stats", "location": "stats/a.go:7",
		"title": "remove unreachable code in stats", "allowed": ".go files directly in stats/ and files under stats/testdata/",
	} {
		var out bytes.Buffer
		if err := run([]string{"describe", "-candidate", cand, "-field", field}, &out); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(out.String()) != want {
			t.Errorf("%s = %q, want %q", field, out.String(), want)
		}
	}
	var out bytes.Buffer
	if err := run([]string{"marker", "-candidate", cand}, &out); err != nil {
		t.Fatal(err)
	}
	// The marker must round-trip through rejectedIDs.
	ids := rejectedIDs([]pullRequest{{Body: out.String()}})
	if !ids["0123456789ab"] {
		t.Errorf("marker %q is not recognized", out.String())
	}
}

func TestUsageErrors(t *testing.T) {
	var usage usageError
	for _, args := range [][]string{
		nil,
		{"bogus"},
		{"select"},
		{"gate", "-open-prs", "x", "extra"},
		{"describe", "-candidate", "x"},
		{"gate", "-open-prs", "x", "-max-open", "0"},
		{"check", "-candidate", "x", "-numstat", "y", "-max-lines", "0"},
		{"round", "-state", "x", "-max", "-1"},
	} {
		if err := run(args, &bytes.Buffer{}); !errors.As(err, &usage) {
			t.Errorf("%v: want usage error, got %v", args, err)
		}
	}
}
