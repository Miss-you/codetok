package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	beginMarker = "BEGIN_ARCHIVE_SUMMARY"
	endMarker   = "END_ARCHIVE_SUMMARY"
)

// Summary is the only thing the AI writes. Every field is required; lists may
// be empty only where noted.
type Summary struct {
	TaskID      string     `json:"task_id"`
	PR          int        `json:"pr"`
	WhatChanged []string   `json:"what_changed"` // non-empty
	Decisions   []Decision `json:"decisions"`    // may be empty
	FollowUps   []FollowUp `json:"follow_ups"`   // may be empty
	Files       []FileRole `json:"files"`        // non-empty, paths from the manifest
}

type Decision struct {
	Decision  string `json:"decision"`
	Rationale string `json:"rationale"`
}

type FollowUp struct {
	Item   string `json:"item"`
	Status string `json:"status"` // todo, handoff or unverified
}

type FileRole struct {
	Path string `json:"path"`
	Role string `json:"role"`
}

var followUpStatuses = map[string]bool{"todo": true, "handoff": true, "unverified": true}

// summarySchema is shown to the model; validateSummary enforces the same rules.
const summarySchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["task_id", "pr", "what_changed", "decisions", "follow_ups", "files"],
  "properties": {
    "task_id": {"type": "string", "description": "exactly the task id given above"},
    "pr": {"type": "integer", "description": "exactly the PR number given above"},
    "what_changed": {"type": "array", "minItems": 1, "items": {"type": "string", "minLength": 1}},
    "decisions": {"type": "array", "items": {"type": "object", "additionalProperties": false,
      "required": ["decision", "rationale"],
      "properties": {"decision": {"type": "string", "minLength": 1}, "rationale": {"type": "string", "minLength": 1}}}},
    "follow_ups": {"type": "array", "items": {"type": "object", "additionalProperties": false,
      "required": ["item", "status"],
      "properties": {"item": {"type": "string", "minLength": 1}, "status": {"enum": ["todo", "handoff", "unverified"]}}}},
    "files": {"type": "array", "minItems": 1, "items": {"type": "object", "additionalProperties": false,
      "required": ["path", "role"],
      "properties": {"path": {"type": "string", "description": "a path from the archived file list"}, "role": {"type": "string", "minLength": 1}}}}
  }
}`

// ExtractSummary returns the JSON between the last BEGIN/END marker pair in
// the agent output. A ```json fence inside the markers is tolerated.
func ExtractSummary(output string) ([]byte, error) {
	end := strings.LastIndex(output, endMarker)
	if end < 0 {
		return nil, fmt.Errorf("agent output has no %s ... %s block", beginMarker, endMarker)
	}
	begin := strings.LastIndex(output[:end], beginMarker)
	if begin < 0 {
		return nil, fmt.Errorf("agent output has %s but no %s before it", endMarker, beginMarker)
	}
	body := strings.TrimSpace(output[begin+len(beginMarker) : end])
	if strings.HasPrefix(body, "```") {
		body = strings.TrimPrefix(body, "```json")
		body = strings.TrimPrefix(body, "```")
		body = strings.TrimSuffix(strings.TrimSpace(body), "```")
	}
	return []byte(strings.TrimSpace(body)), nil
}

// ParseSummary decodes and validates the AI summary against the manifest.
func ParseSummary(data []byte, m Manifest) (Summary, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Summary{}, fmt.Errorf("summary is not a JSON object: %w", err)
	}
	for _, k := range []string{"task_id", "pr", "what_changed", "decisions", "follow_ups", "files"} {
		if v, ok := raw[k]; !ok || string(v) == "null" {
			return Summary{}, fmt.Errorf("summary is missing %q", k)
		}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var s Summary
	if err := dec.Decode(&s); err != nil {
		return Summary{}, fmt.Errorf("summary does not match the schema: %w", err)
	}
	if dec.More() {
		return Summary{}, errors.New("summary has trailing data after the JSON object")
	}
	return s, validateSummary(s, m)
}

func validateSummary(s Summary, m Manifest) error {
	var errs []string
	bad := func(format string, a ...any) { errs = append(errs, fmt.Sprintf(format, a...)) }
	if s.TaskID != m.TaskID {
		bad("task_id is %q, want %q", s.TaskID, m.TaskID)
	}
	if s.PR != m.PR.Number {
		bad("pr is %d, want %d", s.PR, m.PR.Number)
	}
	if len(s.WhatChanged) == 0 {
		bad("what_changed is empty")
	}
	for i, w := range s.WhatChanged {
		if blank(w) {
			bad("what_changed[%d] is empty", i)
		}
	}
	for i, d := range s.Decisions {
		if blank(d.Decision) || blank(d.Rationale) {
			bad("decisions[%d] needs both decision and rationale", i)
		}
	}
	for i, f := range s.FollowUps {
		if blank(f.Item) {
			bad("follow_ups[%d].item is empty", i)
		}
		if !followUpStatuses[f.Status] {
			bad("follow_ups[%d].status is %q, want todo, handoff or unverified", i, f.Status)
		}
	}
	known := map[string]bool{}
	for _, f := range m.Files {
		known[f.Path] = true
	}
	if len(s.Files) == 0 {
		bad("files is empty")
	}
	for i, f := range s.Files {
		if !known[f.Path] {
			bad("files[%d].path %q is not an archived file", i, f.Path)
		}
		if blank(f.Role) {
			bad("files[%d].role is empty", i)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("summary does not match the schema:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func blank(s string) bool { return strings.TrimSpace(s) == "" }

// RenderReadme turns the validated summary into the archive's README.md.
func RenderReadme(s Summary, m Manifest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s process records\n\n", m.TaskID)
	fmt.Fprintf(&b, "- PR: [#%d %s](%s)\n", m.PR.Number, oneLine(m.PR.Title), m.PR.URL)
	fmt.Fprintf(&b, "- Merge commit: `%s`\n", m.PR.MergeCommit)
	if m.PR.MergedAt != "" {
		fmt.Fprintf(&b, "- Merged at: %s\n", m.PR.MergedAt)
	}
	fmt.Fprintf(&b, "- Records comment: %d by @%s\n\n", m.Comment.ID, m.Comment.Author)
	b.WriteString("Generated by `scripts/archive-records.sh` after the merge. The sections below are an AI summary; `raw/` holds verbatim copies of the listed files as of the merge commit. This is history: do not edit it to match later code. See [../AGENTS.md](../AGENTS.md).\n\n")

	b.WriteString("## What changed\n\n")
	for _, w := range s.WhatChanged {
		fmt.Fprintf(&b, "- %s\n", oneLine(w))
	}
	b.WriteString("\n## Key decisions\n\n")
	if len(s.Decisions) == 0 {
		b.WriteString("None recorded.\n")
	}
	for _, d := range s.Decisions {
		fmt.Fprintf(&b, "- **%s**: %s\n", oneLine(d.Decision), oneLine(d.Rationale))
	}
	b.WriteString("\n## Follow-ups and handoffs\n\n")
	if len(s.FollowUps) == 0 {
		b.WriteString("None recorded.\n")
	}
	for _, f := range s.FollowUps {
		fmt.Fprintf(&b, "- [%s] %s\n", f.Status, oneLine(f.Item))
	}
	b.WriteString("\n## Files\n\n| File | Role |\n| --- | --- |\n")
	for _, f := range s.Files {
		fmt.Fprintf(&b, "| [%s](raw/%s) | %s |\n", f.Path, f.Path, strings.ReplaceAll(oneLine(f.Role), "|", "\\|"))
	}
	b.WriteString("\n## Archived files\n\n")
	for _, f := range m.Files {
		fmt.Fprintf(&b, "- [raw/%s](raw/%s) (%s, %d bytes, sha256 `%s`)\n", f.Path, f.Path, f.Source, f.Bytes, f.SHA256[:12])
	}
	return b.String()
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// Finalize validates the agent output and replaces dest/<task>/ in repo with
// the staged archive. Nothing in repo changes when validation fails.
func Finalize(repo, dest, stage, aiOutput string) (Manifest, error) {
	archive := filepath.Join(stage, "archive")
	m, err := readManifest(filepath.Join(archive, "manifest.json"))
	if err != nil {
		return m, fmt.Errorf("staged manifest: %w", err)
	}
	out, err := os.ReadFile(aiOutput)
	if err != nil {
		return m, err
	}
	data, err := ExtractSummary(string(out))
	if err != nil {
		return m, fmt.Errorf("%w; nothing was written", err)
	}
	s, err := ParseSummary(data, m)
	if err != nil {
		return m, fmt.Errorf("%w\nnothing was written; agent output: %s", err, aiOutput)
	}
	if err := writeJSON(filepath.Join(archive, "summary.json"), s); err != nil {
		return m, err
	}
	if err := os.WriteFile(filepath.Join(archive, "README.md"), []byte(RenderReadme(s, m)), 0o644); err != nil {
		return m, err
	}

	root := filepath.Join(repo, filepath.FromSlash(dest))
	target := filepath.Join(root, m.TaskID)
	tmp := filepath.Join(root, "."+m.TaskID+".tmp")
	if err := os.RemoveAll(tmp); err != nil {
		return m, err
	}
	if err := copyTree(archive, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return m, err
	}
	if err := os.RemoveAll(target); err != nil {
		_ = os.RemoveAll(tmp)
		return m, err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.RemoveAll(tmp)
		return m, err
	}
	return m, nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		to := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(to, 0o755)
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s is not a regular file", p)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(to, data, 0o644)
	})
}

// CommitMessage is the archive commit's message. "[skip ci]" keeps push-to-main
// workflows (CI today, any AI review workflow later) from running on it.
func CommitMessage(taskID string, pr int) string {
	return fmt.Sprintf("docs: archive %s records (#%d) [skip ci]", taskID, pr)
}

// buildPrompt gives the agent everything it may use; it never reads the repo.
func buildPrompt(pr PR, m Manifest, contents map[string][]byte, diff []byte) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You are organizing the process records of a merged pull request in the codetok repository (a Go CLI) into a fixed summary.

Rules:
- Summarize only what the material below says. Do not invent decisions, follow-ups or files.
- Do not quote credentials, tokens, account names or private URLs, even if the material contains them.
- Do not run commands or edit files. Everything you need is in this prompt.
- The original files are archived verbatim by a script. You only write the summary.
- Write in English. Keep each string to one or two sentences.

Answer with one JSON object that matches this JSON Schema, between a line %[1]s and a line %[2]s, and nothing after %[2]s:

%[3]s

Use task_id %[4]q and pr %[5]d. In "files", list every archived file below with a short role ("design", "test strategy", "task board row", ...). Use "follow_ups" for open todos, work handed to another task or person, and checks that were not run ("unverified").

`, beginMarker, endMarker, summarySchema, m.TaskID, m.PR.Number)

	fence := "~~~~~~~~"
	fmt.Fprintf(&b, "## Pull request #%d: %s\n\n%s\n%s\n%s\n\n", pr.Number, oneLine(pr.Title), fence, strings.TrimSpace(pr.Body), fence)
	b.WriteString("## Archived files\n\n")
	for _, f := range m.Files {
		data := contents[f.Path]
		note := ""
		if len(data) > promptFileBytes {
			data = truncateUTF8(data, promptFileBytes)
			note = fmt.Sprintf(" (first %d of %d bytes)", len(data), f.Bytes)
		}
		fmt.Fprintf(&b, "### %s%s\n\n%s\n%s\n%s\n\n", f.Path, note, fence, strings.TrimRight(string(data), "\n"), fence)
	}
	if len(diff) > 0 {
		note := ""
		if len(diff) > promptDiffBytes {
			n := len(diff)
			diff = truncateUTF8(diff, promptDiffBytes)
			note = fmt.Sprintf(" (first %d of %d bytes)", len(diff), n)
		}
		fmt.Fprintf(&b, "## Pull request diff%s\n\n%s\n%s\n%s\n", note, fence, strings.TrimRight(string(diff), "\n"), fence)
	}
	return b.String()
}

func truncateUTF8(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	b = b[:n]
	// Drop a rune cut in half by the limit (at most UTFMax-1 bytes).
	for i := 0; i < utf8.UTFMax && len(b) > 0; i++ {
		if r, _ := utf8.DecodeLastRune(b); r != utf8.RuneError {
			break
		}
		b = b[:len(b)-1]
	}
	return b
}
