// Command autooptimize holds the deterministic steps of the auto-optimize
// workflow (docs/auto-optimize.md): collecting candidates from fixed tools,
// picking one, checking the change's scope and size, the open-PR limit, the
// fix-round budget, and reading the reviewer's verdict. scripts/auto-optimize.sh
// runs the tools, the agent and gh; everything that decides goes through here
// so it is covered by `make test`.
//
// Exit status: 0 go on, 10 skip this run without an error (no candidate, an
// open PR exists, budget spent, review rejected, diff out of scope), 1 error,
// 2 usage error.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

const exitSkip = 10

// skipError ends the run without failing it.
type skipError struct{ reason string }

func (e skipError) Error() string { return e.reason }

type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func main() {
	err := run(os.Args[1:], os.Stdout)
	var skip skipError
	var usage usageError
	switch {
	case err == nil:
	case errors.As(err, &skip):
		fmt.Fprintf(os.Stderr, "auto-optimize: skip: %s\n", skip.reason)
		os.Exit(exitSkip)
	case errors.As(err, &usage):
		fmt.Fprintf(os.Stderr, "auto-optimize: %s\n", usage.msg)
		os.Exit(2)
	default:
		fmt.Fprintf(os.Stderr, "auto-optimize: %v\n", err)
		os.Exit(1)
	}
}

const usage = `usage: autooptimize <command> [flags]

commands:
  collect   build the sorted candidate list from tool outputs
  select    pick the first candidate that is not excluded
  describe  print one field of a candidate (id, kind, package, title, allowed, message, location)
  marker    print the PR-body marker for a candidate
  gate      skip when the open optimization PR limit is reached
  check     check a staged diff against the candidate's scope and size cap
  round     take the next fix round, or skip when the budget is spent
  verdict   pass or reject based on the reviewer's output`

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return usageError{usage}
	}
	cmd, args := args[0], args[1:]
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var err error
	switch cmd {
	case "collect":
		err = runCollect(fs, args, stdout)
	case "select":
		err = runSelect(fs, args, stdout)
	case "describe":
		err = runDescribe(fs, args, stdout)
	case "marker":
		err = runMarker(fs, args, stdout)
	case "gate":
		err = runGate(fs, args, stdout)
	case "check":
		err = runCheck(fs, args, stdout)
	case "round":
		err = runRound(fs, args, stdout)
	case "verdict":
		err = runVerdict(fs, args, stdout)
	case "-h", "--help", "help":
		_, _ = fmt.Fprintln(stdout, usage)
		return nil
	default:
		return usageError{fmt.Sprintf("unknown command %q\n%s", cmd, usage)}
	}
	return err
}

func parse(fs *flag.FlagSet, args []string, required ...string) error {
	if err := fs.Parse(args); err != nil {
		return usageError{fmt.Sprintf("%s: %v", fs.Name(), err)}
	}
	if fs.NArg() > 0 {
		return usageError{fmt.Sprintf("%s: unexpected argument %q", fs.Name(), fs.Arg(0))}
	}
	for _, name := range required {
		if fs.Lookup(name).Value.String() == "" {
			return usageError{fmt.Sprintf("%s: -%s is required", fs.Name(), name)}
		}
	}
	return nil
}

func runCollect(fs *flag.FlagSet, args []string, stdout io.Writer) error {
	repo := fs.String("repo", ".", "repository root")
	lint := fs.String("lint", "", "golangci-lint JSON report")
	dead := fs.String("deadcode", "", "deadcode -json report")
	cover := fs.String("cover", "", "go test -cover output")
	minCover := fs.Float64("min-coverage", 60, "report packages below this statement coverage")
	docs := fs.Bool("docs", true, "compare README flags with cmd/ flag definitions")
	out := fs.String("out", "", "write the candidate list here (default stdout)")
	if err := parse(fs, args); err != nil {
		return err
	}
	var all []Candidate
	add := func(file string, parseFn func([]byte) ([]Candidate, error)) error {
		if file == "" {
			return nil
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		cs, err := parseFn(data)
		if err != nil {
			return err
		}
		all = append(all, cs...)
		return nil
	}
	if err := add(*lint, parseLint); err != nil {
		return err
	}
	if err := add(*dead, parseDeadcode); err != nil {
		return err
	}
	if *cover != "" {
		module, err := readModule(*repo)
		if err != nil {
			return err
		}
		if err := add(*cover, func(b []byte) ([]Candidate, error) { return parseCoverage(b, module, *minCover) }); err != nil {
			return err
		}
	}
	if *docs {
		cs, err := collectDocDrift(*repo)
		if err != nil {
			return err
		}
		all = append(all, cs...)
	}
	sortCandidates(all)
	all = dedupe(all)
	if all == nil {
		all = []Candidate{}
	}
	return writeJSON(*out, stdout, all)
}

func runSelect(fs *flag.FlagSet, args []string, stdout io.Writer) error {
	candidates := fs.String("candidates", "", "candidate list from collect")
	exclude := fs.String("exclude", "", "file with candidate IDs to skip, one per line")
	closed := fs.String("closed-prs", "", "gh pr list --state closed --json number,body,mergedAt output")
	out := fs.String("out", "", "write the chosen candidate here (default stdout)")
	if err := parse(fs, args, "candidates"); err != nil {
		return err
	}
	var cs []Candidate
	if err := readJSON(*candidates, &cs); err != nil {
		return err
	}
	excluded := map[string]bool{}
	if *exclude != "" {
		ids, err := readIDList(*exclude)
		if err != nil {
			return err
		}
		excluded = ids
	}
	if *closed != "" {
		prs, err := readPRs(*closed)
		if err != nil {
			return err
		}
		for id := range rejectedIDs(prs) {
			excluded[id] = true
		}
	}
	c, ok := selectCandidate(cs, excluded)
	if !ok {
		return skipError{fmt.Sprintf("no candidate left (%d found, %d excluded)", len(cs), len(excluded))}
	}
	return writeJSON(*out, stdout, c)
}

func runDescribe(fs *flag.FlagSet, args []string, stdout io.Writer) error {
	file := fs.String("candidate", "", "candidate JSON from select")
	field := fs.String("field", "", "id, kind, package, title, allowed, message or location")
	if err := parse(fs, args, "candidate", "field"); err != nil {
		return err
	}
	var c Candidate
	if err := readJSON(*file, &c); err != nil {
		return err
	}
	var v string
	switch *field {
	case "id":
		v = c.ID
	case "kind":
		v = c.Kind
	case "package":
		v = c.Package
	case "title":
		v = c.Title()
	case "allowed":
		v = c.AllowedPaths()
	case "message":
		v = c.Message
	case "location":
		switch {
		case c.File != "" && c.Line > 0:
			v = fmt.Sprintf("%s:%d", c.File, c.Line)
		case c.File != "":
			v = c.File
		default:
			v = c.Package + "/"
		}
	default:
		return usageError{fmt.Sprintf("describe: unknown field %q", *field)}
	}
	_, err := fmt.Fprintln(stdout, v)
	return err
}

func runMarker(fs *flag.FlagSet, args []string, stdout io.Writer) error {
	file := fs.String("candidate", "", "candidate JSON from select")
	if err := parse(fs, args, "candidate"); err != nil {
		return err
	}
	var c Candidate
	if err := readJSON(*file, &c); err != nil {
		return err
	}
	_, err := fmt.Fprintf(stdout, candidateMarkerFormat+"\n", c.ID)
	return err
}

func runGate(fs *flag.FlagSet, args []string, stdout io.Writer) error {
	open := fs.String("open-prs", "", "gh pr list --state open --label <label> --json number,url output")
	maxOpen := fs.Int("max-open", 1, "maximum open optimization PRs")
	if err := parse(fs, args, "open-prs"); err != nil {
		return err
	}
	if *maxOpen < 1 {
		return usageError{"gate: -max-open must be at least 1"}
	}
	prs, err := readPRs(*open)
	if err != nil {
		return err
	}
	if len(prs) >= *maxOpen {
		return skipError{fmt.Sprintf("%d optimization PR(s) still open (limit %d), first: #%d %s",
			len(prs), *maxOpen, prs[0].Number, prs[0].URL)}
	}
	_, err = fmt.Fprintf(stdout, "%d open optimization PR(s), limit %d\n", len(prs), *maxOpen)
	return err
}

func runCheck(fs *flag.FlagSet, args []string, stdout io.Writer) error {
	file := fs.String("candidate", "", "candidate JSON from select")
	numstat := fs.String("numstat", "", "git diff --cached --numstat --no-renames <base> output")
	maxLines := fs.Int("max-lines", 200, "maximum added+deleted lines")
	if err := parse(fs, args, "candidate", "numstat"); err != nil {
		return err
	}
	var c Candidate
	if err := readJSON(*file, &c); err != nil {
		return err
	}
	data, err := os.ReadFile(*numstat)
	if err != nil {
		return err
	}
	entries, err := parseNumstat(data)
	if err != nil {
		return err
	}
	if problems := checkDiff(c, entries, *maxLines); len(problems) > 0 {
		for _, p := range problems {
			_, _ = fmt.Fprintf(stdout, "- %s\n", p)
		}
		return skipError{fmt.Sprintf("diff violates the scope rules (%d problem(s))", len(problems))}
	}
	_, err = fmt.Fprintf(stdout, "diff ok: %d file(s) within %s\n", len(entries), c.AllowedPaths())
	return err
}

func runRound(fs *flag.FlagSet, args []string, stdout io.Writer) error {
	state := fs.String("state", "", "round counter file for this run")
	maxRounds := fs.Int("max", 2, "maximum fix rounds")
	if err := parse(fs, args, "state"); err != nil {
		return err
	}
	n, ok, err := nextRound(*state, *maxRounds)
	if err != nil {
		return err
	}
	if !ok {
		return skipError{fmt.Sprintf("fix-round budget spent (%d of %d used)", n, *maxRounds)}
	}
	_, err = fmt.Fprintln(stdout, n)
	return err
}

func runVerdict(fs *flag.FlagSet, args []string, stdout io.Writer) error {
	file := fs.String("review", "", "reviewer output")
	if err := parse(fs, args, "review"); err != nil {
		return err
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	ok, reason := reviewVerdict(string(data))
	if !ok {
		return skipError{"review rejected: " + reason}
	}
	_, err = fmt.Fprintf(stdout, "review passed: %s\n", reason)
	return err
}

func readJSON(file string, v any) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", file, err)
	}
	return nil
}

func writeJSON(file string, stdout io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if file == "" {
		_, err = stdout.Write(data)
		return err
	}
	return os.WriteFile(file, data, 0o644)
}
