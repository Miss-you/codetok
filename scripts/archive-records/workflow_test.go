package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The archive workflow must not feed back into itself or into AI review, must
// always run its file from main (never from a PR, so a fork cannot rewrite it),
// and must only run for the repository owner. The repository has no YAML
// dependency, so these checks read the file as text; keep the layout simple.
func TestArchiveWorkflowGuards(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/archive-records.yml")
	if err != nil {
		t.Fatal(err)
	}
	wf := string(data)
	onBlock := regexp.MustCompile(`(?ms)^on:\n(.*?)^\S`).FindStringSubmatch(wf)
	if onBlock == nil {
		t.Fatal("no top-level on: block")
	}
	var triggers []string
	for _, line := range strings.Split(onBlock[1], "\n") {
		if m := regexp.MustCompile(`^  ([a-z_]+):`).FindStringSubmatch(line); m != nil {
			triggers = append(triggers, m[1])
		}
	}
	if strings.Join(triggers, ",") != "push,workflow_dispatch" {
		t.Errorf("triggers = %v, want only push and workflow_dispatch (never pull_request, pull_request_target or issue_comment)", triggers)
	}
	if !strings.Contains(onBlock[1], "  push:\n    branches: [main]\n") {
		t.Errorf("push trigger must be limited to branches: [main]:\n%s", onBlock[1])
	}
	for _, want := range []string{
		"github.repository == 'Miss-you/codetok'",
		"github.triggering_actor == github.repository_owner",
		"!contains(github.event.head_commit.message, '[skip ci]')",
		"scripts/archive-records.sh --commit \"$GITHUB_SHA\"",
		"runs-on: [self-hosted, codetok-local, ai]",
		"persist-credentials: false",
		"contents: read",
		"scripts/archive-records.sh --pr \"$PR_NUMBER\"",
	} {
		if !strings.Contains(wf, want) {
			t.Errorf("workflow is missing %q", want)
		}
	}
	// Third-party actions on the self-hosted runner are pinned by commit SHA.
	for _, m := range regexp.MustCompile(`(?m)^\s*-?\s*uses:\s*(\S+)`).FindAllStringSubmatch(wf, -1) {
		if !regexp.MustCompile(`@[0-9a-f]{40}$`).MatchString(m[1]) {
			t.Errorf("action %s is not pinned by commit SHA", m[1])
		}
	}
	for _, banned := range []string{"pull_request:", "pull_request_target", "issue_comment", "github.event.pull_request", "secrets.", "github.event.comment"} {
		if strings.Contains(wf, banned) {
			t.Errorf("workflow must not contain %q", banned)
		}
	}
}
