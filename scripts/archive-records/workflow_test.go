package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The archive workflow must not feed back into itself or into AI review, and
// must never run for fork PRs. The repository has no YAML dependency, so these
// checks read the file as text; keep the workflow's layout simple.
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
	if strings.Join(triggers, ",") != "pull_request,workflow_dispatch" {
		t.Errorf("triggers = %v, want only pull_request and workflow_dispatch (never push, pull_request_target or issue_comment)", triggers)
	}
	for _, want := range []string{
		"types: [closed]",
		"branches: [main]",
		"github.event.pull_request.merged == true",
		"github.event.pull_request.head.repo.full_name == github.repository",
		"github.repository == 'Miss-you/codetok'",
		"runs-on: [self-hosted, codetok-local, ai]",
		"persist-credentials: false",
		"contents: read",
		"scripts/archive-records.sh --pr \"$PR_NUMBER\"",
	} {
		if !strings.Contains(wf, want) {
			t.Errorf("workflow is missing %q", want)
		}
	}
	for _, banned := range []string{"pull_request_target", "issue_comment", "secrets.", "github.event.pull_request.title", "github.event.pull_request.body", "github.event.comment"} {
		if strings.Contains(wf, banned) {
			t.Errorf("workflow must not contain %q", banned)
		}
	}
}
