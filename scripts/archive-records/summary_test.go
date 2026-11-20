package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testManifest() Manifest {
	return Manifest{
		TaskID: "ENG-012",
		PR:     ManifestPR{Number: 57, Title: "ci: archive", URL: "https://example.invalid/pull/57", MergeCommit: "abc123"},
		Files: []ManifestFile{
			{Path: "workspace/ENG-012/final_impl.md", Source: "repo", SHA256: strings.Repeat("a", 64), Bytes: 10},
		},
		InputDigest: strings.Repeat("d", 64),
	}
}

const validSummary = `{"task_id":"ENG-012","pr":57,
 "what_changed":["Added the archive workflow."],
 "decisions":[{"decision":"Archive to docs/changes","rationale":"Not every task has an OpenSpec change."}],
 "follow_ups":[{"item":"Run on the runner","status":"unverified"}],
 "files":[{"path":"workspace/ENG-012/final_impl.md","role":"final design"}]}`

func TestExtractSummary(t *testing.T) {
	out := "[client] initialize (running)\nThinking about BEGIN_ARCHIVE_SUMMARY usage\n" +
		beginMarker + "\n```json\n" + validSummary + "\n```\n" + endMarker + "\n\n[done] end_turn\n"
	data, err := ExtractSummary(out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseSummary(data, testManifest()); err != nil {
		t.Fatalf("ParseSummary: %v\n%s", err, data)
	}
	if _, err := ExtractSummary("no markers here"); err == nil {
		t.Fatal("want an error without markers")
	}
	if _, err := ExtractSummary("x " + endMarker); err == nil {
		t.Fatal("want an error with END but no BEGIN")
	}
}

func TestParseSummaryRejects(t *testing.T) {
	cases := map[string]string{
		"not json":         `{"task_id":`,
		"unknown field":    strings.Replace(validSummary, `"pr":57,`, `"pr":57,"extra":1,`, 1),
		"missing field":    strings.Replace(validSummary, `"follow_ups":[{"item":"Run on the runner","status":"unverified"}],`, ``, 1),
		"null field":       strings.Replace(validSummary, `"decisions":[{"decision":"Archive to docs/changes","rationale":"Not every task has an OpenSpec change."}]`, `"decisions":null`, 1),
		"wrong task":       strings.Replace(validSummary, `"ENG-012"`, `"ENG-999"`, 1),
		"wrong pr":         strings.Replace(validSummary, `"pr":57`, `"pr":58`, 1),
		"empty changes":    strings.Replace(validSummary, `["Added the archive workflow."]`, `[]`, 1),
		"blank decision":   strings.Replace(validSummary, `"rationale":"Not every task has an OpenSpec change."`, `"rationale":"  "`, 1),
		"bad status":       strings.Replace(validSummary, `"unverified"`, `"done"`, 1),
		"unknown file":     strings.Replace(validSummary, `"path":"workspace/ENG-012/final_impl.md"`, `"path":"cmd/root.go"`, 1),
		"trailing data":    validSummary + ` {}`,
		"wrong type":       strings.Replace(validSummary, `"pr":57`, `"pr":"57"`, 1),
		"nested unknown":   strings.Replace(validSummary, `"role":"final design"`, `"role":"final design","why":"x"`, 1),
		"no files":         strings.Replace(validSummary, `[{"path":"workspace/ENG-012/final_impl.md","role":"final design"}]`, `[]`, 1),
		"empty whats item": strings.Replace(validSummary, `["Added the archive workflow."]`, `[""]`, 1),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseSummary([]byte(in), testManifest()); err == nil {
				t.Fatalf("want an error for\n%s", in)
			}
		})
	}
}

func TestFinalizeWritesOnlyValidOutput(t *testing.T) {
	repo := t.TempDir()
	stage := t.TempDir()
	m := testManifest()
	raw := filepath.Join(stage, "archive", "raw", "workspace", "ENG-012")
	if err := os.MkdirAll(raw, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raw, "final_impl.md"), []byte("verbatim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(stage, "archive", "manifest.json"), m); err != nil {
		t.Fatal(err)
	}
	// An older archive with a stale file that must disappear on overwrite.
	old := filepath.Join(repo, "docs", "changes", "ENG-012")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "stale.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	bad := filepath.Join(stage, "bad.txt")
	if err := os.WriteFile(bad, []byte(beginMarker+"\n{\"task_id\":\"ENG-012\"}\n"+endMarker), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Finalize(repo, "docs/changes", stage, bad); err == nil || !strings.Contains(err.Error(), "nothing was written") {
		t.Fatalf("invalid output: err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(old, "stale.md")); err != nil {
		t.Fatalf("invalid output changed the existing archive: %v", err)
	}

	good := filepath.Join(stage, "good.txt")
	if err := os.WriteFile(good, []byte(beginMarker+"\n"+validSummary+"\n"+endMarker+"\n[done] end_turn\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Finalize(repo, "docs/changes", stage, good); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"README.md", "summary.json", "manifest.json", "raw/workspace/ENG-012/final_impl.md"} {
		if _, err := os.Stat(filepath.Join(old, filepath.FromSlash(f))); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(old, "stale.md")); !os.IsNotExist(err) {
		t.Errorf("stale file survived the overwrite: %v", err)
	}
	if entries, _ := filepath.Glob(filepath.Join(repo, "docs", "changes", ".*.tmp")); len(entries) != 0 {
		t.Errorf("temporary directory left behind: %v", entries)
	}
	readme, _ := os.ReadFile(filepath.Join(old, "README.md"))
	for _, want := range []string{"# ENG-012 process records", "[#57 ci: archive]", "**Archive to docs/changes**", "[unverified] Run on the runner", "(raw/workspace/ENG-012/final_impl.md)"} {
		if !strings.Contains(string(readme), want) {
			t.Errorf("README missing %q:\n%s", want, readme)
		}
	}
}

func TestCommitMessageSkipsCI(t *testing.T) {
	got := CommitMessage("ENG-012", 57)
	if got != "docs: archive ENG-012 records (#57) [skip ci]" {
		t.Fatalf("CommitMessage = %q", got)
	}
}
