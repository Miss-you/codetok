package main

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// The records comment is the only handoff between the delivery flow and the
// post-merge archive. Its format is deliberately small:
//
//	<!-- codetok-records:ENG-012 -->
//	### Process records: ENG-012
//	...
//	- `workspace/ENG-012/`
//	- `docs/plans/2026-06-13-engineering-roadmap.md`
//
//	<details><summary>inline: notes.md</summary>
//
//	~~~~text
//	verbatim content that is not committed to the repository
//	~~~~
//
//	</details>
//
// Bullets of the form "- `path`" outside inline blocks list repository paths;
// a trailing "/" means "every file under this directory". Inline blocks carry
// content that never reached the repository.

const markerPrefix = "codetok-records:"

var (
	// The marker counts only on a line of its own, so quoting it inline in a
	// review or discussion comment (`<!-- codetok-records:X -->`) does not.
	markerRe      = regexp.MustCompile(`(?m)^<!-- codetok-records:([^ >\r\n]*) -->[ \t]*\r?$`)
	taskIDRe      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	inlineNameRe  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
	bulletRe      = regexp.MustCompile("^\\s*[-*] `([^`]+)`")
	summaryLineRe = regexp.MustCompile(`^\s*<details><summary>inline: ([^<]*)</summary>\s*$`)
)

// allowedPrefixes are the only repository locations a records comment may
// list. Anything else (source code, .github, docs/changes itself) is rejected.
var allowedPrefixes = []string{"workspace/", "openspec/changes/", "docs/plans/"}

// trustedAssociations are the GitHub author_association values whose marker
// comments are honored. On a public repository anyone can comment, so a marker
// from anyone else is ignored rather than archived.
var trustedAssociations = map[string]bool{"OWNER": true, "MEMBER": true, "COLLABORATOR": true}

// Records is the parsed content of one records comment.
type Records struct {
	TaskID string
	Paths  []string
	Inline []InlineFile
}

// InlineFile is a process file carried verbatim inside the comment.
type InlineFile struct {
	Name    string
	Content string
}

// Marker returns the hidden marker line for a task.
func Marker(taskID string) string {
	return "<!-- " + markerPrefix + taskID + " -->"
}

// RenderComment builds the records comment body. ParseComment(RenderComment(r))
// returns r, with inline content normalized to end in a newline.
func RenderComment(r Records) (string, error) {
	if err := validateRecords(r); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(Marker(r.TaskID) + "\n")
	fmt.Fprintf(&b, "### Process records: %s\n\n", r.TaskID)
	fmt.Fprintf(&b, "After this PR merges, `scripts/archive-records.sh` copies these files verbatim into `docs/changes/%s/raw/` on `main` and adds an AI summary. Edit this comment when the list changes; do not post a second one.\n\n", r.TaskID)
	if len(r.Paths) > 0 {
		b.WriteString("Files:\n")
		for _, p := range r.Paths {
			fmt.Fprintf(&b, "- `%s`\n", p)
		}
	}
	for _, f := range r.Inline {
		content := f.Content
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		fence := strings.Repeat("~", max(4, longestRun(content, '~')+1))
		fmt.Fprintf(&b, "\n<details><summary>inline: %s</summary>\n\n%stext\n%s%s\n\n</details>\n", f.Name, fence, content, fence)
	}
	return b.String(), nil
}

// ParseComment extracts the records from a comment body. ok is false when the
// body has no marker; an error means the marker is present but the comment is
// malformed, which the archive treats as a failure rather than a silent skip.
func ParseComment(body string) (r Records, ok bool, err error) {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	markers := markerRe.FindAllStringSubmatch(body, -1)
	if len(markers) == 0 {
		return Records{}, false, nil
	}
	if len(markers) > 1 {
		return Records{}, true, fmt.Errorf("comment has %d records markers, want exactly one", len(markers))
	}
	r.TaskID = markers[0][1]

	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if m := summaryLineRe.FindStringSubmatch(line); m != nil {
			f, next, err := parseInline(lines, i+1, m[1])
			if err != nil {
				return Records{}, true, err
			}
			r.Inline = append(r.Inline, f)
			i = next
			continue
		}
		if m := bulletRe.FindStringSubmatch(line); m != nil {
			r.Paths = append(r.Paths, m[1])
		}
	}
	if err := validateRecords(r); err != nil {
		return Records{}, true, err
	}
	return r, true, nil
}

// parseInline reads one fenced block starting at lines[start] and returns the
// index of the line holding </details>.
func parseInline(lines []string, start int, name string) (InlineFile, int, error) {
	i := start
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) {
		return InlineFile{}, 0, fmt.Errorf("inline %q: missing fenced block", name)
	}
	opener := strings.TrimSpace(lines[i])
	fenceChar := opener[0]
	n := 0
	for n < len(opener) && opener[n] == fenceChar {
		n++
	}
	if (fenceChar != '~' && fenceChar != '`') || n < 3 {
		return InlineFile{}, 0, fmt.Errorf("inline %q: expected a ~~~ or ``` fence, got %q", name, opener)
	}
	var content []string
	for i++; i < len(lines); i++ {
		t := strings.TrimRight(lines[i], " \t")
		if len(t) >= n && strings.Trim(t, string(fenceChar)) == "" {
			break
		}
		content = append(content, lines[i])
	}
	if i >= len(lines) {
		return InlineFile{}, 0, fmt.Errorf("inline %q: fence is not closed", name)
	}
	for i++; i < len(lines) && strings.TrimSpace(lines[i]) == ""; i++ {
	}
	if i >= len(lines) || strings.TrimSpace(lines[i]) != "</details>" {
		return InlineFile{}, 0, fmt.Errorf("inline %q: expected </details> after the fence", name)
	}
	text := strings.Join(content, "\n")
	if len(content) > 0 {
		text += "\n"
	}
	return InlineFile{Name: name, Content: text}, i, nil
}

func validateRecords(r Records) error {
	if !taskIDRe.MatchString(r.TaskID) {
		return fmt.Errorf("task id %q: use letters, digits, '.', '_' or '-' (max 64)", r.TaskID)
	}
	if len(r.Paths) == 0 && len(r.Inline) == 0 {
		return fmt.Errorf("records for %s list no files", r.TaskID)
	}
	seen := map[string]bool{}
	for _, p := range r.Paths {
		if err := validateRepoPath(p); err != nil {
			return err
		}
		if seen[p] {
			return fmt.Errorf("path %q is listed twice", p)
		}
		seen[p] = true
	}
	names := map[string]bool{}
	for _, f := range r.Inline {
		if !inlineNameRe.MatchString(f.Name) || f.Name == "." || f.Name == ".." {
			return fmt.Errorf("inline name %q: use a plain file name (letters, digits, '.', '_', '-')", f.Name)
		}
		if names[f.Name] {
			return fmt.Errorf("inline name %q is used twice", f.Name)
		}
		names[f.Name] = true
	}
	return nil
}

// validateRepoPath accepts clean, relative paths under allowedPrefixes. A
// trailing "/" marks a directory.
func validateRepoPath(p string) error {
	trimmed := strings.TrimSuffix(p, "/")
	switch {
	case p == "" || trimmed == "":
		return fmt.Errorf("empty path")
	case strings.ContainsAny(p, "\\\x00*?[") || strings.HasPrefix(p, ":"):
		return fmt.Errorf("path %q: wildcards, backslashes and pathspec magic are not allowed", p)
	case path.IsAbs(p) || path.Clean(trimmed) != trimmed:
		return fmt.Errorf("path %q: must be a clean relative path without '..' or '.'", p)
	}
	for _, seg := range strings.Split(trimmed, "/") {
		if seg == ".." || seg == "." {
			return fmt.Errorf("path %q: must not contain '..' or '.'", p)
		}
	}
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(trimmed+"/", prefix) && trimmed+"/" != prefix {
			return nil
		}
	}
	return fmt.Errorf("path %q: only paths under %s can be archived", p, strings.Join(allowedPrefixes, ", "))
}

// Comment is the subset of a GitHub issue comment the archive reads.
type Comment struct {
	ID                int64  `json:"id"`
	Body              string `json:"body"`
	AuthorAssociation string `json:"author_association"`
	UpdatedAt         string `json:"updated_at"`
	User              struct {
		Login string `json:"login"`
	} `json:"user"`
}

// FindRecords returns the single trusted records comment among comments.
// found is false when no trusted comment carries a marker; markers from
// untrusted authors are reported in ignored so the caller can log them.
func FindRecords(comments []Comment) (r Records, c Comment, found bool, ignored []string, err error) {
	var hits []Comment
	for _, cm := range comments {
		if !markerRe.MatchString(cm.Body) {
			continue
		}
		if !trustedAssociations[cm.AuthorAssociation] {
			ignored = append(ignored, fmt.Sprintf("comment %d by %s (%s)", cm.ID, cm.User.Login, cm.AuthorAssociation))
			continue
		}
		hits = append(hits, cm)
	}
	if len(hits) == 0 {
		return Records{}, Comment{}, false, ignored, nil
	}
	if len(hits) > 1 {
		ids := make([]string, len(hits))
		for i, h := range hits {
			ids[i] = fmt.Sprint(h.ID)
		}
		sort.Strings(ids)
		return Records{}, Comment{}, true, ignored, fmt.Errorf("found %d records comments (%s); keep one and edit it instead of posting another", len(hits), strings.Join(ids, ", "))
	}
	r, _, err = ParseComment(hits[0].Body)
	if err != nil {
		return Records{}, hits[0], true, ignored, fmt.Errorf("records comment %d: %w", hits[0].ID, err)
	}
	return r, hits[0], true, ignored, nil
}

func longestRun(s string, ch byte) int {
	best, cur := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			cur++
			best = max(best, cur)
		} else {
			cur = 0
		}
	}
	return best
}
