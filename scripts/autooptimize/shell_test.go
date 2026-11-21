package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// These tests source scripts/auto-optimize.sh and call its safety helpers
// directly. give_up, die and log are replaced so a failing check prints a
// marker and exits 3 instead of resetting the checkout.

const shellPrelude = `
source "$SCRIPT"
set +e
log() { :; }
give_up() { echo "GIVE_UP: $1"; exit 3; }
die() { echo "DIE: $1"; exit 4; }
`

func runShell(t *testing.T, dir, body string, env ...string) (string, int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("bash helpers are only used on the macOS runner")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	script, err := filepath.Abs(filepath.Join("..", "auto-optimize.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "-c", shellPrelude+body)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), append([]string{"SCRIPT=" + script}, env...)...)
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), code
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.email=t@example.com", "-c", "user.name=t"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// tempRepo makes a git repo with one commit: go.mod, cmd/a.go, .gitignore.
func tempRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	writeFile(t, dir, "go.mod", "module x\n")
	writeFile(t, dir, ".gitignore", "go.work\n")
	if err := os.Mkdir(filepath.Join(dir, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "cmd/a.go", "package cmd\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "base")
	return dir
}

func TestShellNoSecrets(t *testing.T) {
	dir := t.TempDir()
	// Build token-like strings at run time so the repository holds none.
	fake := map[string]string{
		"gh token":   "ghp_" + strings.Repeat("A1", 18),
		"sk key":     "key: sk-" + strings.Repeat("b2", 12),
		"aws key":    "AKIA" + strings.Repeat("Z", 16),
		"private":    "-----BEGIN RSA " + "PRIVATE KEY-----",
		"line start": "sk-" + strings.Repeat("c", 24),
	}
	for name, text := range fake {
		f := writeFile(t, dir, "secret.md", "before\n"+text+"\nafter\n")
		out, code := runShell(t, dir, `no_secrets "$F"`, "F="+f)
		if code != 3 || !strings.Contains(out, "GIVE_UP") {
			t.Errorf("%s: not caught (code %d, %q)", name, code, out)
		}
	}
	clean := writeFile(t, dir, "clean.md",
		"Ran task-auto-optimize-candidate-selection and disk-cache-implementation-notes.\n"+
			"See ask-for-confirmation-first-before-merging and github.com/Miss-you/codetok.\n")
	if out, code := runShell(t, dir, `no_secrets "$F"`, "F="+clean); code != 0 {
		t.Errorf("false positive on prose: code %d, %q", code, out)
	}
}

func TestShellReviewAnswer(t *testing.T) {
	dir := t.TempDir()
	repo := "/work/codetok"
	home := "/Users/someone"
	if err := os.MkdirAll(filepath.Join(dir, "review"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "review/review.md", strings.Join([]string{
		"[client] initialize (running)",
		"[tool] Read file (completed)",
		"  input: {\"Path\":\"" + home + "/.config/gh/hosts.yml\"}",
		"### Strengths",
		"- Clean change in [cmd/a.go](file://" + repo + "/cmd/a.go#L3).",
		"- Also read " + home + "/other/project/x.go.",
		"[tool] Read file (completed)",
		"  output:",
		"    secret file text",
		"",
		"    more secret text",
		"#### Critical (Must Fix)",
		"None.",
		"[done] end_turn",
	}, "\n")+"\n")
	out, code := runShell(t, dir, `out=$PWD repo_root=`+repo+` HOME=`+home+` review_answer`)
	if code != 0 {
		t.Fatalf("code %d: %s", code, out)
	}
	if !strings.HasPrefix(out, "### Strengths\n") {
		t.Errorf("answer does not start at the first heading: %q", out)
	}
	for _, bad := range []string{"[tool]", "[done]", "input:", "output:", "hosts.yml", "secret", repo, home} {
		if strings.Contains(out, bad) {
			t.Errorf("answer contains %q:\n%s", bad, out)
		}
	}
	if !strings.Contains(out, "[cmd/a.go](cmd/a.go#L3)") || !strings.Contains(out, "~/other/project/x.go") {
		t.Errorf("paths not rewritten:\n%s", out)
	}

	// A review with bold titles only still produces an answer, and an empty
	// answer does not fail under pipefail.
	writeFile(t, dir, "review/review.md", "noise\n**Critical (Must Fix)**\nNone\n")
	if out, code := runShell(t, dir, `out=$PWD repo_root=/r review_answer`); code != 0 || !strings.HasPrefix(out, "**Critical") {
		t.Errorf("bold-title review: code %d, %q", code, out)
	}
	writeFile(t, dir, "review/review.md", "[done] end_turn\n")
	if out, code := runShell(t, dir, `set -o pipefail; out=$PWD repo_root=/r review_answer; echo rc=$?`); code != 0 || !strings.Contains(out, "rc=0") {
		t.Errorf("empty review: code %d, %q", code, out)
	}
}

func TestShellRevertOutOfScope(t *testing.T) {
	repo := tempRepo(t)
	out := t.TempDir()
	writeFile(t, repo, "go.mod", "module x\nrequire y v1\n")
	writeFile(t, repo, "cmd/a.go", "package cmd\n\nvar keep = 1\n")
	writeFile(t, repo, "cmd/new note.md", "stray\n")
	git(t, repo, "add", "-A")
	writeFile(t, out, "out-of-scope.txt", "go.mod\ncmd/new note.md\n")

	res, code := runShell(t, repo, `out=$OUT repo_root=$PWD base=$(git rev-parse HEAD); : >"$out/failure.md"; revert_out_of_scope; git status --porcelain`, "OUT="+out)
	if code != 0 {
		t.Fatalf("code %d: %s", code, res)
	}
	if strings.TrimSpace(res) != "M  cmd/a.go" {
		t.Errorf("status after revert = %q, want only the in-scope edit", res)
	}
	if data, _ := os.ReadFile(filepath.Join(repo, "go.mod")); string(data) != "module x\n" {
		t.Errorf("go.mod not restored: %q", data)
	}
	if fail, _ := os.ReadFile(filepath.Join(out, "failure.md")); !strings.Contains(string(fail), "- cmd/new note.md") {
		t.Errorf("failure.md does not tell the agent: %q", fail)
	}
}

func TestShellGuardDetectsAndUndoesTampering(t *testing.T) {
	repo := tempRepo(t)
	out := t.TempDir()
	ao := writeFile(t, out, "autooptimize", "binary")
	writeFile(t, out, "candidate.json", `{"id":"x"}`)
	setup := `out=$OUT repo_root=$PWD ao=$AO; guard_snapshot; `

	// No change: the check passes.
	if res, code := runShell(t, repo, setup+`guard_check; echo clean`, "OUT="+out, "AO="+ao); code != 0 || !strings.Contains(res, "clean") {
		t.Fatalf("untouched checkout failed the guard: code %d, %s", code, res)
	}

	tamper := map[string]string{
		"hook":          `mkdir -p .git/hooks; printf '#!/bin/sh\necho pwned\n' > .git/hooks/pre-commit`,
		"git config":    `git config core.hooksPath /tmp/evil`,
		"ignored file":  `echo 'go 1.24' > go.work`,
		"run state":     `echo '{"id":"y","kind":"docs"}' > "$out/candidate.json"`,
		"info/excludes": `mkdir -p .git/info; echo 'cmd/' >> .git/info/exclude`,
	}
	for name, change := range tamper {
		t.Run(name, func(t *testing.T) {
			repo := tempRepo(t)
			out := t.TempDir()
			ao := writeFile(t, out, "autooptimize", "binary")
			writeFile(t, out, "candidate.json", `{"id":"x"}`)
			res, code := runShell(t, repo, `out=$OUT repo_root=$PWD ao=$AO; guard_snapshot; `+change+`; guard_check; echo missed`,
				"OUT="+out, "AO="+ao)
			if code != 3 || !strings.Contains(res, "GIVE_UP") {
				t.Fatalf("tampering not detected: code %d, %s", code, res)
			}
			if _, err := os.Stat(filepath.Join(repo, ".git/hooks/pre-commit")); err == nil {
				t.Error("planted hook survived")
			}
			if _, err := os.Stat(filepath.Join(repo, "go.work")); err == nil {
				t.Error("planted ignored file survived")
			}
			cfg, _ := os.ReadFile(filepath.Join(repo, ".git/config"))
			if strings.Contains(string(cfg), "hooksPath") {
				t.Error("git config change survived")
			}
			if ex, _ := os.ReadFile(filepath.Join(repo, ".git/info/exclude")); strings.Contains(string(ex), "cmd/") {
				t.Error("exclude change survived")
			}
		})
	}
}

func TestShellRunAgentRequiresAFinishedTurn(t *testing.T) {
	repo := tempRepo(t)
	out := t.TempDir()
	ao := writeFile(t, out, "autooptimize", "binary")
	writeFile(t, out, "candidate.json", `{"id":"x"}`)
	writeFile(t, out, "prompt.md", "do it")
	tests := []struct {
		name   string
		log    string
		status int
		want   int
	}{
		{"finished", `printf '[tool] x\nDone.\n[done] end_turn\n'`, 0, 0},
		{"exit 0 after an API error", `printf '⚠ API error (attempt 3)\n'`, 0, 1},
		{"denied but finished", `printf '[tool] grep (failed)\n[done] end_turn\n\n'`, 5, 0},
		{"denied and stopped", `printf '[tool] grep (failed)\n'`, 5, 5},
		{"timeout", `printf 'partial\n[done] end_turn\n'`, 3, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `out=$OUT repo_root=$PWD ao=$AO model=m agent_timeout=1
agy() { :; }
acpx() { ` + tt.log + `; return ` + strconv.Itoa(tt.status) + `; }
run_agent "$out/prompt.md" "$out/agent.log"; echo "rc=$?"`
			res, code := runShell(t, repo, body, "OUT="+out, "AO="+ao)
			if code != 0 || !strings.Contains(res, "rc="+strconv.Itoa(tt.want)) {
				t.Errorf("code %d, output %q, want rc=%d", code, res, tt.want)
			}
		})
	}
}

func TestShellNoSecretsRemovesFlaggedFiles(t *testing.T) {
	dir := t.TempDir()
	f := writeFile(t, dir, "pr-body.md", "token ghp_"+strings.Repeat("Q9", 18)+"\n")
	g := writeFile(t, dir, "review-comment.md", "clean\n")
	if _, code := runShell(t, dir, `no_secrets "$F" "$G"`, "F="+f, "G="+g); code != 3 {
		t.Fatalf("code %d, want give_up", code)
	}
	for _, p := range []string{f, g} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s still exists and would be uploaded", filepath.Base(p))
		}
	}
}

func TestShellNoSecretsInCommit(t *testing.T) {
	repo := tempRepo(t)
	run := func(content string) int {
		writeFile(t, repo, "cmd/a.go", content)
		git(t, repo, "add", "-A")
		git(t, repo, "commit", "-q", "-m", "change")
		_, code := runShell(t, repo, `base=$(git rev-parse HEAD~1) head=$(git rev-parse HEAD); no_secrets_in_commit; echo ok`)
		return code
	}
	if code := run("package cmd\n\n// task-auto-optimize-candidate-selection-notes\n"); code != 0 {
		t.Errorf("clean change: code %d", code)
	}
	if code := run("package cmd\n\nconst k = \"ghp_" + strings.Repeat("Z7", 18) + "\"\n"); code != 3 {
		t.Errorf("added token: code %d, want give_up", code)
	}
}
