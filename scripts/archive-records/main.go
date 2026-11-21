// Command archive-records holds the deterministic parts of the post-merge
// archive of task process records (GitHub issue #57). scripts/archive-records.sh
// drives it; scripts/post-records-comment.sh uses the comment subcommand.
//
//	archive-records comment --task ID [--inline NAME=FILE]... [PATH]...
//	archive-records plan --pr-json F --comments F --repo DIR --tree SHA --dest DIR --stage DIR [--diff F]
//	archive-records finalize --repo DIR --dest DIR --stage DIR --ai-output F
//
// plan and finalize print key=value lines on stdout for the shell script.
// The AI never writes into the repository: plan copies files verbatim from the
// merge commit, and finalize only adds summary.json and README.md after the
// agent output passes validation.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "archive-records: %v\n", err)
		os.Exit(1)
	}
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: archive-records comment|plan|finalize [flags]")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	switch args[0] {
	case "comment":
		task := fs.String("task", "", "task id, e.g. ENG-012")
		var inline multiFlag
		fs.Var(&inline, "inline", "NAME=FILE: attach FILE verbatim as an inline block (repeatable)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		r := Records{TaskID: *task, Paths: fs.Args()}
		for _, spec := range inline {
			name, file, ok := strings.Cut(spec, "=")
			if !ok {
				return fmt.Errorf("--inline %q: want NAME=FILE", spec)
			}
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			r.Inline = append(r.Inline, InlineFile{Name: name, Content: string(data)})
		}
		body, err := RenderComment(r)
		if err != nil {
			return err
		}
		if len(body) > 65000 {
			return fmt.Errorf("comment is %d characters; GitHub allows 65536. Commit large files under workspace/%s/ and list them instead of inlining", len(body), r.TaskID)
		}
		_, err = io.WriteString(stdout, body)
		return err

	case "plan":
		prJSON := fs.String("pr-json", "", "output of gh pr view --json number,title,body,url,state,baseRefName,isCrossRepository,mergeCommit,mergedAt")
		commentsJSON := fs.String("comments", "", "output of gh api --paginate repos/OWNER/REPO/issues/N/comments")
		repo := fs.String("repo", ".", "git work tree that receives the archive")
		tree := fs.String("tree", "", "commit to read the listed files from (default: the PR merge commit)")
		dest := fs.String("dest", "docs/changes", "archive root, relative to --repo")
		stage := fs.String("stage", "", "empty directory for the staged archive")
		diff := fs.String("diff", "", "optional PR diff for the prompt")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *prJSON == "" || *commentsJSON == "" || *stage == "" {
			return errors.New("plan needs --pr-json, --comments and --stage")
		}
		var pr PR
		if err := readJSONFile(*prJSON, &pr); err != nil {
			return err
		}
		comments, err := readComments(*commentsJSON)
		if err != nil {
			return err
		}
		res, err := Plan(planInput{PR: pr, Comments: comments, Repo: *repo, Tree: *tree, Dest: *dest, Stage: *stage, DiffPath: *diff})
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "status=%s\ntask_id=%s\nreason=%s\n", res.Status, res.TaskID, res.Reason)
		return nil

	case "finalize":
		repo := fs.String("repo", ".", "git work tree that receives the archive")
		dest := fs.String("dest", "docs/changes", "archive root, relative to --repo")
		stage := fs.String("stage", "", "directory written by plan")
		aiOutput := fs.String("ai-output", "", "agent output holding the summary JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *stage == "" || *aiOutput == "" {
			return errors.New("finalize needs --stage and --ai-output")
		}
		m, err := Finalize(*repo, *dest, *stage, *aiOutput)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "task_id=%s\ncommit_message=%s\n", m.TaskID, CommitMessage(m.TaskID, m.PR.Number))
		return nil
	}
	return fmt.Errorf("unknown subcommand %q", args[0])
}

func readJSONFile(p string, v any) error {
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	return nil
}

// readComments accepts what `gh api --paginate` prints: one JSON array per
// page, concatenated, or a stream of single comment objects.
func readComments(p string) ([]Comment, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var out []Comment
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err == io.EOF {
			return out, nil
		} else if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		if strings.HasPrefix(strings.TrimSpace(string(raw)), "[") {
			var page []Comment
			if err := json.Unmarshal(raw, &page); err != nil {
				return nil, fmt.Errorf("%s: %w", p, err)
			}
			out = append(out, page...)
			continue
		}
		var c Comment
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		out = append(out, c)
	}
}
