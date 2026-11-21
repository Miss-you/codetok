package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitRepo creates a repository with files committed and returns its path and
// the commit SHA.
func gitRepo(t *testing.T, files map[string]string) (string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "user.name", "test")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "commit.gpgsign", "false")
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", "init")
	return dir, strings.TrimSpace(runGit(t, dir, "rev-parse", "HEAD"))
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

var repoFiles = map[string]string{
	"workspace/ENG-012/final_impl.md":    "# final\n\nverbatim  spacing\t kept\n",
	"workspace/ENG-012/sub/notes.md":     "notes\n",
	"docs/plans/roadmap.md":              "| row |\n",
	"workspace/OTHER/secret-ish.md":      "not listed\n",
	"cmd/root.go":                        "package cmd\n",
	"openspec/changes/foo/proposal.md":   "proposal\n",
	"openspec/changes/foo/specs/a/s.md":  "spec\n",
	"workspace/ENG-012/big-but-fine.txt": strings.Repeat("x", 1000),
}

func mergedPR(sha string) PR {
	pr := PR{Number: 57, Title: "ci: archive", Body: "Closes #57", URL: "https://github.com/Miss-you/codetok/pull/57", State: "MERGED", BaseRefName: "main", MergedAt: "2026-11-20T00:00:00Z"}
	pr.MergeCommit = &struct {
		OID string `json:"oid"`
	}{OID: sha}
	return pr
}

func recordsComment(t *testing.T, r Records) []Comment {
	t.Helper()
	body, err := RenderComment(r)
	if err != nil {
		t.Fatal(err)
	}
	return []Comment{comment(1, "OWNER", "unrelated"), comment(42, "OWNER", body)}
}

func TestPlanCopiesVerbatim(t *testing.T) {
	repo, sha := gitRepo(t, repoFiles)
	stage := t.TempDir()
	comments := recordsComment(t, Records{
		TaskID: "ENG-012",
		Paths:  []string{"workspace/ENG-012/", "docs/plans/roadmap.md", "openspec/changes/foo/", "workspace/ENG-012/final_impl.md"},
		Inline: []InlineFile{{Name: "session.md", Content: "inline body\n"}},
	})
	res, err := Plan(planInput{PR: mergedPR(sha), Comments: comments, Repo: repo, Dest: "docs/changes", Stage: stage})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "archive" || res.TaskID != "ENG-012" {
		t.Fatalf("res = %+v", res)
	}
	wantFiles := map[string]string{
		"workspace/ENG-012/final_impl.md":    repoFiles["workspace/ENG-012/final_impl.md"],
		"workspace/ENG-012/sub/notes.md":     repoFiles["workspace/ENG-012/sub/notes.md"],
		"workspace/ENG-012/big-but-fine.txt": repoFiles["workspace/ENG-012/big-but-fine.txt"],
		"docs/plans/roadmap.md":              repoFiles["docs/plans/roadmap.md"],
		"openspec/changes/foo/proposal.md":   repoFiles["openspec/changes/foo/proposal.md"],
		"openspec/changes/foo/specs/a/s.md":  repoFiles["openspec/changes/foo/specs/a/s.md"],
		"inline/session.md":                  "inline body\n",
	}
	var got []string
	_ = filepath.WalkDir(filepath.Join(stage, "archive", "raw"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(filepath.Join(stage, "archive", "raw"), p)
			got = append(got, filepath.ToSlash(rel))
		}
		return nil
	})
	if len(got) != len(wantFiles) {
		t.Fatalf("raw files = %v, want %d files", got, len(wantFiles))
	}
	for rel, want := range wantFiles {
		data, err := os.ReadFile(filepath.Join(stage, "archive", "raw", filepath.FromSlash(rel)))
		if err != nil || string(data) != want {
			t.Errorf("raw/%s = %q, %v; want verbatim %q", rel, data, err, want)
		}
	}
	m, err := readManifest(filepath.Join(stage, "archive", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if m.PR.MergeCommit != sha || m.Comment.ID != 42 || len(m.Files) != len(wantFiles) {
		t.Fatalf("manifest = %+v", m)
	}
	prompt, err := os.ReadFile(filepath.Join(stage, "prompt.md"))
	if err != nil || !strings.Contains(string(prompt), "### workspace/ENG-012/final_impl.md") || !strings.Contains(string(prompt), beginMarker) {
		t.Fatalf("prompt missing content: %v", err)
	}
}

func TestPlanSkips(t *testing.T) {
	repo, sha := gitRepo(t, repoFiles)
	good := recordsComment(t, Records{TaskID: "ENG-012", Paths: []string{"workspace/ENG-012/"}})
	fork := mergedPR(sha)
	fork.IsCrossRepository = true
	open := mergedPR(sha)
	open.State, open.MergeCommit = "CLOSED", nil
	other := mergedPR(sha)
	other.BaseRefName = "release"
	cases := []struct {
		name     string
		pr       PR
		comments []Comment
		reason   string
	}{
		{"fork", fork, good, "fork"},
		{"not merged", open, good, "not merged"},
		{"other base", other, good, "targets release"},
		{"no records comment", mergedPR(sha), []Comment{comment(1, "OWNER", "AI review: x")}, "no records comment"},
		{"untrusted records comment", mergedPR(sha), []Comment{comment(1, "NONE", Marker("ENG-012")+"\n- `workspace/ENG-012/`")}, "no records comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stage := t.TempDir()
			res, err := Plan(planInput{PR: tc.pr, Comments: tc.comments, Repo: repo, Dest: "docs/changes", Stage: stage})
			if err != nil || res.Status != "skip" || !strings.Contains(res.Reason, tc.reason) {
				t.Fatalf("res=%+v err=%v, want skip containing %q", res, err, tc.reason)
			}
			if entries, _ := os.ReadDir(stage); len(entries) != 0 {
				t.Fatalf("stage not empty after skip: %v", entries)
			}
		})
	}
}

func TestPlanErrors(t *testing.T) {
	files := map[string]string{"workspace/A/a.md": "a\n", "workspace/A/dir/b.md": "b\n"}
	repo, _ := gitRepo(t, files)
	// Add a symlink in a second commit.
	if err := os.Symlink("../../cmd", filepath.Join(repo, "workspace", "A", "link")); err != nil {
		t.Skip("symlinks unsupported:", err)
	}
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "link")
	sha := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	cases := map[string][]string{
		"missing file":          {"workspace/A/nope.md"},
		"dir without slash":     {"workspace/A/dir"},
		"file with slash":       {"workspace/A/a.md/"},
		"symlink in listed dir": {"workspace/A/"},
	}
	for name, paths := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Plan(planInput{PR: mergedPR(sha), Comments: recordsComment(t, Records{TaskID: "A", Paths: paths}), Repo: repo, Dest: "docs/changes", Stage: t.TempDir()})
			if err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

func TestPlanIdempotentDigest(t *testing.T) {
	repo, sha := gitRepo(t, repoFiles)
	comments := recordsComment(t, Records{TaskID: "ENG-012", Paths: []string{"workspace/ENG-012/"}})
	stage := t.TempDir()
	if _, err := Plan(planInput{PR: mergedPR(sha), Comments: comments, Repo: repo, Dest: "docs/changes", Stage: stage}); err != nil {
		t.Fatal(err)
	}
	// Pretend the archive landed: copy the staged manifest to docs/changes/ENG-012/.
	dst := filepath.Join(repo, "docs", "changes", "ENG-012")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(stage, "archive", "manifest.json"))
	if err := os.WriteFile(filepath.Join(dst, "manifest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Same inputs, even with a different comment id and author: skip.
	again := recordsComment(t, Records{TaskID: "ENG-012", Paths: []string{"workspace/ENG-012/"}})
	again[1].ID = 99
	res, err := Plan(planInput{PR: mergedPR(sha), Comments: again, Repo: repo, Dest: "docs/changes", Stage: t.TempDir()})
	if err != nil || res.Status != "skip" || !strings.Contains(res.Reason, "already holds") {
		t.Fatalf("rerun: res=%+v err=%v", res, err)
	}

	// Different inputs (an extra inline file): archive again.
	changed := recordsComment(t, Records{TaskID: "ENG-012", Paths: []string{"workspace/ENG-012/"}, Inline: []InlineFile{{Name: "x.md", Content: "x"}}})
	res, err = Plan(planInput{PR: mergedPR(sha), Comments: changed, Repo: repo, Dest: "docs/changes", Stage: t.TempDir()})
	if err != nil || res.Status != "archive" {
		t.Fatalf("changed inputs: res=%+v err=%v", res, err)
	}
}

func TestReadCommentsPaginated(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.json")
	page1, _ := json.Marshal([]Comment{comment(1, "OWNER", "a")})
	page2, _ := json.Marshal([]Comment{comment(2, "OWNER", "b"), comment(3, "NONE", "c")})
	if err := os.WriteFile(p, append(append(page1, '\n'), page2...), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readComments(p)
	if err != nil || len(got) != 3 || got[2].AuthorAssociation != "NONE" {
		t.Fatalf("got %+v, %v", got, err)
	}
}
