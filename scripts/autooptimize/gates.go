package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// candidateMarker is written into every optimization PR body so later runs can
// tell which candidate a closed PR proposed.
const candidateMarkerFormat = "<!-- auto-optimize-candidate: %s -->"

var candidateMarker = regexp.MustCompile(`<!-- auto-optimize-candidate: ([0-9a-f]{12}) -->`)

type pullRequest struct {
	Number   int     `json:"number"`
	URL      string  `json:"url"`
	Body     string  `json:"body"`
	MergedAt *string `json:"mergedAt"`
}

func readPRs(file string) ([]pullRequest, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var prs []pullRequest
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, fmt.Errorf("parse %s: %w", file, err)
	}
	return prs, nil
}

// rejectedIDs returns the candidates proposed by PRs that were closed without
// being merged: a person already said no to them.
func rejectedIDs(prs []pullRequest) map[string]bool {
	ids := map[string]bool{}
	for _, pr := range prs {
		if pr.MergedAt != nil && *pr.MergedAt != "" {
			continue
		}
		for _, m := range candidateMarker.FindAllStringSubmatch(pr.Body, -1) {
			ids[m[1]] = true
		}
	}
	return ids
}

// readIDList reads one candidate ID per line; blank lines and # comments are
// ignored and anything after the first field is a free-form note.
func readIDList(file string) (map[string]bool, error) {
	ids := map[string]bool{}
	f, err := os.Open(file)
	if os.IsNotExist(err) {
		return ids, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ids[strings.Fields(line)[0]] = true
	}
	return ids, sc.Err()
}

// selectCandidate returns the first candidate whose ID is not excluded.
func selectCandidate(cs []Candidate, excluded map[string]bool) (Candidate, bool) {
	for _, c := range cs {
		if !excluded[c.ID] {
			return c, true
		}
	}
	return Candidate{}, false
}

// diffEntry is one line of `git diff --numstat --no-renames`.
type diffEntry struct {
	Added, Deleted int
	Binary         bool
	Path           string
}

func parseNumstat(data []byte) ([]diffEntry, error) {
	var out []diffEntry
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.SplitN(line, "\t", 3)
		if len(f) != 3 {
			return nil, fmt.Errorf("bad numstat line %q", line)
		}
		e := diffEntry{Path: filepath.ToSlash(f[2])}
		if f[0] == "-" || f[1] == "-" {
			e.Binary = true
		} else {
			a, err1 := strconv.Atoi(f[0])
			d, err2 := strconv.Atoi(f[1])
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("bad numstat line %q", line)
			}
			e.Added, e.Deleted = a, d
		}
		out = append(out, e)
	}
	return out, nil
}

// checkDiff enforces the scope rules for one candidate: a size cap, no
// excluded paths, no binaries, and only files of the candidate's package (or
// only the READMEs for a docs candidate). It returns every violation found.
func checkDiff(c Candidate, entries []diffEntry, maxLines int) []string {
	problems, _ := checkDiffPaths(c, entries, maxLines)
	return problems
}

// checkDiffPaths is checkDiff that also returns the paths that may not be
// changed at all, so the caller can put them back.
func checkDiffPaths(c Candidate, entries []diffEntry, maxLines int) (problems, outOfScope []string) {
	if len(entries) == 0 {
		return []string{"the change is empty"}, nil
	}
	total := 0
	for _, e := range entries {
		total += e.Added + e.Deleted
		problem := ""
		switch {
		case e.Binary:
			problem = fmt.Sprintf("%s: binary files are not allowed", e.Path)
		case isExcludedPath(e.Path):
			problem = fmt.Sprintf("%s: this path is never changed by auto-optimize", e.Path)
		case !pathAllowed(c, e.Path):
			problem = fmt.Sprintf("%s: outside the allowed scope (%s)", e.Path, c.AllowedPaths())
		}
		if problem != "" {
			problems = append(problems, problem)
			outOfScope = append(outOfScope, e.Path)
		}
	}
	if total > maxLines {
		problems = append(problems, fmt.Sprintf("diff is %d changed lines, the cap is %d", total, maxLines))
	}
	return problems, outOfScope
}

func pathAllowed(c Candidate, p string) bool {
	if c.Kind == KindDocs {
		for _, d := range docFiles {
			if p == d {
				return true
			}
		}
		return false
	}
	if c.Package == "" {
		return false
	}
	if strings.HasPrefix(p, path.Join(c.Package, "testdata")+"/") {
		return true
	}
	if pkgDir(p) != c.Package || !strings.HasSuffix(p, ".go") {
		return false
	}
	if c.Kind == KindCoverage {
		return strings.HasSuffix(p, "_test.go")
	}
	return true
}

// nextRound increments the fix-round counter stored in file and reports
// whether another fix round is allowed.
func nextRound(file string, maxRounds int) (int, bool, error) {
	n := 0
	data, err := os.ReadFile(file)
	switch {
	case err == nil:
		n, err = strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return 0, false, fmt.Errorf("bad round counter in %s: %w", file, err)
		}
	case !os.IsNotExist(err):
		return 0, false, err
	}
	if n >= maxRounds {
		return n, false, nil
	}
	n++
	if err := os.WriteFile(file, []byte(strconv.Itoa(n)+"\n"), 0o644); err != nil {
		return 0, false, err
	}
	return n, true, nil
}

var (
	verdictLine = regexp.MustCompile(`(?i)ready to merge\?\**\s*:?\s*\**\s*(yes|no|with fixes)`)
	headingLine = regexp.MustCompile(`^#{1,6}\s+(.*)$`)
	boldLine    = regexp.MustCompile(`^\*\*([^*]+)\*\*:?$`) // a whole line in bold, used as a title
	itemLine    = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+(.*)$`)
	// acpx trace lines around the answer: tool calls, their input/output, and
	// the end-of-turn marker. They are never part of the review.
	traceLine = regexp.MustCompile(`^(\[(client|tool|done|thinking)\]|  (input|output|kind|files):)`)
	noneItem  = regexp.MustCompile(`(?i)^(none|n/a|nothing|no issues?)( found)?\.?$`)
)

// sectionFor maps a title to "critical", "important", "other" (a known
// section that ends the issue list) or "" (not a section title).
func sectionFor(title string) string {
	// Section titles start with the severity ("Critical (Must Fix)"); a
	// title that only mentions it ("Summary of critical paths") does not count.
	t := strings.ToLower(strings.Trim(strings.TrimSpace(title), "*_ "))
	switch {
	case strings.HasPrefix(t, "critical"):
		return "critical"
	case strings.HasPrefix(t, "important"):
		return "important"
	}
	// Only the template's own section titles end an issue list; a bold
	// sub-title such as "Summary of problem" inside Critical does not.
	for _, other := range []string{"minor", "nice to have", "strength", "recommendation", "assessment", "issues"} {
		if strings.HasPrefix(t, other) {
			return "other"
		}
	}
	return ""
}

// reviewVerdict decides whether the independent review accepts the change.
// It passes only when the reviewer's verdict is not "No", both the Critical
// and the Important sections are present, and neither lists an issue. Any
// text in those sections other than "None" counts as at least one issue, so
// free-form findings are not missed.
func reviewVerdict(review string) (bool, string) {
	verdict := ""
	sections := map[string][]string{}
	seen := map[string]bool{}
	section, inFence, inTrace := "", false, false
	for _, line := range strings.Split(review, "\n") {
		// An acpx trace block is a "[tool] ..." (or [client]/[thinking]) line
		// followed by indented lines with its input and output, which can
		// quote file contents. None of it is the reviewer's answer.
		if traceLine.MatchString(line) {
			inTrace = true
			continue
		}
		if inTrace && (strings.HasPrefix(line, "  ") || strings.TrimSpace(line) == "") {
			continue
		}
		inTrace = false
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
		}
		if !inFence {
			if v := verdictLine.FindStringSubmatch(line); v != nil {
				verdict = strings.ToLower(v[1]) // the last verdict outside traces and fences wins
				section = ""                    // a verdict line ends the issue list even without a heading
				continue
			}
			title := ""
			if h := headingLine.FindStringSubmatch(trimmed); h != nil {
				title = h[1]
			} else if b := boldLine.FindStringSubmatch(trimmed); b != nil {
				title = b[1]
			}
			if next := sectionFor(title); title != "" && next != "" {
				section = next
				if next != "other" {
					seen[next] = true
				} else {
					section = ""
				}
				continue
			}
		}
		if section != "" && trimmed != "" {
			sections[section] = append(sections[section], line)
		}
	}
	if verdict == "" {
		return false, "the review has no \"Ready to merge?\" verdict"
	}
	if !seen["critical"] || !seen["important"] {
		return false, "the review has no Critical and Important sections to check"
	}
	blocking := 0
	for _, lines := range sections {
		items, text := 0, 0
		for _, line := range lines {
			body := strings.TrimSpace(line)
			if it := itemLine.FindStringSubmatch(line); it != nil {
				body = it[1]
			}
			if noneItem.MatchString(strings.Trim(body, "*_ \t")) {
				continue
			}
			text++
			// Top-level bullets and numbered items are issues; indented
			// bullets describe the issue above them.
			if itemLine.MatchString(line) && (strings.TrimLeft(line, " \t") == line || itemIsNumbered(line)) {
				items++
			}
		}
		if items == 0 && text > 0 {
			items = 1
		}
		blocking += items
	}
	switch {
	case verdict == "no":
		return false, "the reviewer answered \"Ready to merge? No\""
	case blocking > 0:
		return false, fmt.Sprintf("the reviewer listed %d Critical/Important issue(s)", blocking)
	}
	return true, fmt.Sprintf("the reviewer answered %q with no Critical/Important issues", verdict)
}

func itemIsNumbered(line string) bool {
	t := strings.TrimLeft(line, " \t")
	return len(t) > 0 && t[0] >= '0' && t[0] <= '9'
}
