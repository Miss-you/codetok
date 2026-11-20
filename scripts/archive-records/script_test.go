package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// scriptEnv runs scripts/archive-records.sh against a local bare "origin" with
// stub gh, acpx and agy commands, so the whole post-merge flow (skip rules,
// verbatim copy, idempotent rerun, commit message, push) is tested without
// GitHub or a model.
type scriptEnv struct {
	t      *testing.T
	origin string
	clone  string
	stubs  string
	tool   string
	merge  string
}

func newScriptEnv(t *testing.T) *scriptEnv {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("archive-records.sh runs on the macOS runner")
	}
	for _, bin := range []string{"bash", "git", "python3"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not installed", bin)
		}
	}
	src, _ := gitRepo(t, map[string]string{
		"workspace/ENG-012/final_impl.md": "# final\n",
		"workspace/ENG-012/todo.md":       "- runner check\n",
		"docs/changes/AGENTS.md":          "# archive\n",
	})
	e := &scriptEnv{t: t, origin: filepath.Join(t.TempDir(), "origin.git"), clone: filepath.Join(t.TempDir(), "clone"), stubs: t.TempDir()}
	runGit(t, src, "clone", "-q", "--bare", src, e.origin)
	runGit(t, src, "clone", "-q", e.origin, e.clone)
	e.merge = e.originHead()

	e.tool = filepath.Join(t.TempDir(), "archive-records")
	if out, err := exec.Command("go", "build", "-o", e.tool, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	e.stub("gh", `#!/bin/bash
set -e
d=$(dirname "$0")
case "$1 $2" in
  "pr view") cat "$d/pr-$3.json" ;;
  "api repos/"*/pulls) cat "$d/pulls.json" ;;
  "api --paginate") n=$(sed -E 's#.*/issues/([0-9]+)/comments#\1#' <<<"$3"); cat "$d/comments-$n.json" ;;
  *) echo "unexpected gh call: $*" >&2; exit 9 ;;
esac
`)
	e.stub("acpx", `#!/bin/bash
d=$(dirname "$0")
echo call >>"$d/acpx-calls"
for a in "$@"; do [[ $a == --deny-all ]] && denied=1; [[ $a == --no-terminal ]] && noterm=1; done
[[ ${denied:-0} == 1 && ${noterm:-0} == 1 ]] || { echo "acpx must run with --deny-all --no-terminal" >&2; exit 8; }
echo "[client] initialize (running)"
cat "$d/ai-output.txt"
echo "[done] end_turn"
`)
	e.stub("agy", "#!/bin/bash\nexit 0\n")
	return e
}

func (e *scriptEnv) stub(name, body string) {
	if err := os.WriteFile(filepath.Join(e.stubs, name), []byte(body), 0o755); err != nil {
		e.t.Fatal(err)
	}
}

func (e *scriptEnv) originHead() string {
	return strings.TrimSpace(runGit(e.t, e.origin, "rev-parse", "refs/heads/main"))
}

func (e *scriptEnv) acpxCalls() int {
	data, _ := os.ReadFile(filepath.Join(e.stubs, "acpx-calls"))
	return strings.Count(string(data), "call")
}

func (e *scriptEnv) setPR(n int, cross bool, comments []Comment) {
	pr := mergedPR(e.merge)
	pr.Number, pr.IsCrossRepository = n, cross
	data, _ := json.Marshal(pr)
	if err := os.WriteFile(filepath.Join(e.stubs, "pr-"+itoa(n)+".json"), data, 0o644); err != nil {
		e.t.Fatal(err)
	}
	cdata, _ := json.Marshal(comments)
	if err := os.WriteFile(filepath.Join(e.stubs, "comments-"+itoa(n)+".json"), cdata, 0o644); err != nil {
		e.t.Fatal(err)
	}
}

func (e *scriptEnv) setAI(out string) {
	if err := os.WriteFile(filepath.Join(e.stubs, "ai-output.txt"), []byte(out), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

func (e *scriptEnv) run(n int) (string, error) {
	return e.runArgs("--pr", itoa(n))
}

// setPulls sets what GitHub answers for repos/.../commits/<sha>/pulls.
func (e *scriptEnv) setPulls(body string) {
	if err := os.WriteFile(filepath.Join(e.stubs, "pulls.json"), []byte(body), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

func (e *scriptEnv) runArgs(args ...string) (string, error) {
	script, _ := filepath.Abs("../archive-records.sh")
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir = e.clone
	cmd.Env = append(os.Environ(),
		"PATH="+e.stubs+string(os.PathListSeparator)+os.Getenv("PATH"),
		"AI_REVIEW_PROXY=none", "ARCHIVE_RECORDS_BIN="+e.tool, "RUNNER_TEMP="+e.t.TempDir(),
		"GIT_AUTHOR_NAME=archive", "GIT_AUTHOR_EMAIL=a@example.com", "GIT_COMMITTER_NAME=archive", "GIT_COMMITTER_EMAIL=a@example.com",
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestArchiveScriptEndToEnd(t *testing.T) {
	e := newScriptEnv(t)
	records := recordsComment(t, Records{TaskID: "ENG-012", Paths: []string{"workspace/ENG-012/"}, Inline: []InlineFile{{Name: "session.md", Content: "from the session\n"}}})
	summary := strings.Replace(validSummary, `"files":[{"path":"workspace/ENG-012/final_impl.md","role":"final design"}]`,
		`"files":[{"path":"workspace/ENG-012/final_impl.md","role":"final design"},{"path":"inline/session.md","role":"session notes"}]`, 1)
	e.setPR(57, false, records)
	e.setAI(beginMarker + "\n" + summary + "\n" + endMarker + "\n")
	before := e.originHead()

	out, err := e.run(57)
	if err != nil {
		t.Fatalf("archive run failed: %v\n%s", err, out)
	}
	after := e.originHead()
	if after == before {
		t.Fatalf("no commit pushed:\n%s", out)
	}
	msg := strings.TrimSpace(runGit(t, e.origin, "log", "-1", "--format=%s", "main"))
	if msg != "docs: archive ENG-012 records (#57) [skip ci]" {
		t.Fatalf("commit message = %q", msg)
	}
	changed := strings.Fields(runGit(t, e.origin, "diff", "--name-only", before, after))
	for _, f := range changed {
		if !strings.HasPrefix(f, "docs/changes/ENG-012/") {
			t.Errorf("archive commit touched %s outside docs/changes/ENG-012/", f)
		}
	}
	for path, want := range map[string]string{
		"docs/changes/ENG-012/raw/workspace/ENG-012/final_impl.md": "# final\n",
		"docs/changes/ENG-012/raw/workspace/ENG-012/todo.md":       "- runner check\n",
		"docs/changes/ENG-012/raw/inline/session.md":               "from the session\n",
	} {
		if got := runGit(t, e.origin, "show", "main:"+path); got != want {
			t.Errorf("%s = %q, want verbatim %q", path, got, want)
		}
	}
	runGit(t, e.origin, "cat-file", "-e", "main:docs/changes/ENG-012/README.md")
	runGit(t, e.origin, "cat-file", "-e", "main:docs/changes/ENG-012/summary.json")
	if e.acpxCalls() != 1 {
		t.Fatalf("acpx calls = %d, want 1", e.acpxCalls())
	}
	if wt := runGit(t, e.clone, "worktree", "list"); strings.Count(wt, "\n") != 1 {
		t.Errorf("temporary worktree left behind:\n%s", wt)
	}

	t.Run("rerun is a no-op", func(t *testing.T) {
		// A new summary would differ; the rerun must not even ask for one.
		e.setAI(beginMarker + "\n" + strings.Replace(summary, "Added the archive workflow.", "Something else.", 1) + "\n" + endMarker)
		out, err := e.run(57)
		if err != nil || e.originHead() != after || e.acpxCalls() != 1 || !strings.Contains(out, "already holds") {
			t.Fatalf("rerun: err=%v head changed=%v acpx=%d\n%s", err, e.originHead() != after, e.acpxCalls(), out)
		}
	})

	t.Run("push of a merge commit finds its PR", func(t *testing.T) {
		// Same PR as above, found from the push sha: the digest makes it a no-op.
		e.setPulls(`[{"number":99,"merge_commit_sha":"` + strings.Repeat("0", 40) + `","merged_at":"2026-11-20T00:00:00Z","base":{"ref":"main"}},
		           {"number":57,"merge_commit_sha":"` + e.merge + `","merged_at":"2026-11-20T00:00:00Z","base":{"ref":"main"}}]`)
		out, err := e.runArgs("--commit", e.merge)
		if err != nil || e.originHead() != after || e.acpxCalls() != 1 || !strings.Contains(out, "merge of PR #57") || !strings.Contains(out, "already holds") {
			t.Fatalf("err=%v\n%s", err, out)
		}
	})

	t.Run("push that is not a PR merge is skipped", func(t *testing.T) {
		for name, pulls := range map[string]string{
			"direct push":  `[]`,
			"open PR":      `[{"number":57,"merge_commit_sha":"` + e.merge + `","merged_at":null,"base":{"ref":"main"}}]`,
			"other branch": `[{"number":57,"merge_commit_sha":"` + e.merge + `","merged_at":"2026-11-20T00:00:00Z","base":{"ref":"dev"}}]`,
			"other sha":    `[{"number":57,"merge_commit_sha":"` + strings.Repeat("1", 40) + `","merged_at":"2026-11-20T00:00:00Z","base":{"ref":"main"}}]`,
		} {
			e.setPulls(pulls)
			out, err := e.runArgs("--commit", after)
			if err != nil || e.originHead() != after || !strings.Contains(out, "skipped") {
				t.Fatalf("%s: err=%v\n%s", name, err, out)
			}
		}
		if out, err := e.runArgs("--commit", "abc"); err == nil || !strings.Contains(out, "full commit sha") {
			t.Fatalf("short sha accepted: %v\n%s", err, out)
		}
		if out, err := e.runArgs("--commit", after, "--pr", "57"); err == nil || !strings.Contains(out, "usage") {
			t.Fatalf("--commit with --pr accepted: %v\n%s", err, out)
		}
	})

	t.Run("PR without records comment is skipped", func(t *testing.T) {
		e.setPR(58, false, []Comment{comment(1, "OWNER", "AI review: abc")})
		out, err := e.run(58)
		if err != nil || e.originHead() != after || !strings.Contains(out, "no records comment") {
			t.Fatalf("err=%v\n%s", err, out)
		}
	})

	t.Run("fork PR is skipped", func(t *testing.T) {
		e.setPR(59, true, recordsComment(t, Records{TaskID: "FORK-1", Paths: []string{"workspace/ENG-012/"}}))
		out, err := e.run(59)
		if err != nil || e.originHead() != after || !strings.Contains(out, "fork") {
			t.Fatalf("err=%v\n%s", err, out)
		}
	})

	t.Run("invalid AI output fails without writing", func(t *testing.T) {
		e.setPR(60, false, recordsComment(t, Records{TaskID: "ENG-060", Paths: []string{"workspace/ENG-012/todo.md"}}))
		e.setAI("I could not do it.\n")
		out, err := e.run(60)
		if err == nil || e.originHead() != after || !strings.Contains(out, "nothing was written") {
			t.Fatalf("err=%v head changed=%v\n%s", err, e.originHead() != after, out)
		}
	})
}

// TestPostRecordsCommentScript runs scripts/post-records-comment.sh in this
// repository with a stub gh. The jq filter itself runs inside gh and is not
// exercised here; the stub returns the ids it would select.
func TestPostRecordsCommentScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("post-records-comment.sh is a bash script")
	}
	for _, bin := range []string{"bash", "git", "python3", "go"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not installed", bin)
		}
	}
	if out, err := exec.Command("git", "ls-files", "--error-unmatch", "../../workspace/ENG-012/final_impl.md").CombinedOutput(); err != nil {
		t.Skipf("workspace/ENG-012 is not tracked in this checkout: %s", out)
	}
	stubs := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubs, "gh"), []byte(`#!/bin/bash
d=$(dirname "$0")
echo "$*" >>"$d/calls"
case "$*" in
  *--paginate*) cat "$d/ids" ;;
  "api -X "*)
    while [[ $# -gt 0 ]]; do [[ $1 == --input ]] && cp "$2" "$d/payload.json"; shift; done
    echo https://example.invalid/comment ;;
  *"--jq .body") cat "$d/existing" ;;
  *) echo "unexpected gh call: $*" >&2; exit 9 ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}
	script, _ := filepath.Abs("../post-records-comment.sh")
	run := func(ids, existing string, args ...string) (string, string, error) {
		_ = os.WriteFile(filepath.Join(stubs, "ids"), []byte(ids), 0o644)
		_ = os.WriteFile(filepath.Join(stubs, "existing"), []byte(existing), 0o644)
		_ = os.Remove(filepath.Join(stubs, "calls"))
		_ = os.Remove(filepath.Join(stubs, "payload.json"))
		cmd := exec.Command("/bin/bash", append([]string{script, "--pr", "60", "--task", "ENG-012"}, args...)...)
		cmd.Env = append(os.Environ(), "PATH="+stubs+string(os.PathListSeparator)+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		calls, _ := os.ReadFile(filepath.Join(stubs, "calls"))
		return string(out), string(calls), err
	}
	ok := []string{"workspace/ENG-012/", "docs/plans/2026-06-13-engineering-roadmap.md"}

	t.Run("no paths", func(t *testing.T) {
		if out, _, err := run("", ""); err == nil || !strings.Contains(out, "list at least one path") {
			t.Fatalf("err=%v\n%s", err, out)
		}
	})
	for name, bad := range map[string]string{
		"directory without slash": "workspace/ENG-012",
		"file with slash":         "workspace/ENG-012/final_impl.md/",
		"untracked path":          "workspace/ENG-012/does-not-exist.md",
	} {
		t.Run(name, func(t *testing.T) {
			out, calls, err := run("", "", bad)
			if err == nil || calls != "" {
				t.Fatalf("err=%v gh calls=%q\n%s", err, calls, out)
			}
		})
	}
	t.Run("posts a new comment", func(t *testing.T) {
		out, calls, err := run("", "", ok...)
		if err != nil || !strings.Contains(calls, "api -X POST repos/Miss-you/codetok/issues/60/comments") {
			t.Fatalf("err=%v calls=%q\n%s", err, calls, out)
		}
		var payload struct{ Body string }
		data, _ := os.ReadFile(filepath.Join(stubs, "payload.json"))
		if err := json.Unmarshal(data, &payload); err != nil {
			t.Fatal(err)
		}
		r, found, err := ParseComment(payload.Body)
		if !found || err != nil || r.TaskID != "ENG-012" || len(r.Paths) != 2 {
			t.Fatalf("posted body does not parse: %+v %v %v", r, found, err)
		}
	})
	t.Run("edits the existing comment", func(t *testing.T) {
		out, calls, err := run("123\n", Marker("ENG-012")+" \r\nold\n", ok...)
		if err != nil || !strings.Contains(calls, "api -X PATCH repos/Miss-you/codetok/issues/comments/123") || strings.Contains(calls, "POST") {
			t.Fatalf("err=%v calls=%q\n%s", err, calls, out)
		}
	})
	t.Run("refuses another task's comment", func(t *testing.T) {
		out, calls, err := run("123\n", Marker("ENG-999")+"\nsee `"+Marker("ENG-012")+"`\n", ok...)
		if err == nil || strings.Contains(calls, "-X ") {
			t.Fatalf("err=%v calls=%q\n%s", err, calls, out)
		}
	})
	t.Run("inline file is relative to the caller", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("from the caller dir\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("/bin/bash", script, "--dry-run", "--pr", "60", "--task", "ENG-012", "--inline", "notes.md=notes.md", "workspace/ENG-012/")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), "from the caller dir") {
			t.Fatalf("err=%v\n%s", err, out)
		}
	})
	t.Run("refuses two comments", func(t *testing.T) {
		out, calls, err := run("1\n2\n", "", ok...)
		if err == nil || strings.Contains(calls, "-X ") || !strings.Contains(out, "2 records comments") {
			t.Fatalf("err=%v calls=%q\n%s", err, calls, out)
		}
	})
}
