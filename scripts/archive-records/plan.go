package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxFileBytes      = 2 << 20  // one archived file
	maxTotalBytes     = 20 << 20 // all archived files of one PR
	promptFileBytes   = 48 << 10 // per file, in the AI prompt only
	promptDiffBytes   = 160 << 10
	defaultBaseBranch = "main"
)

// PR is the subset of `gh pr view --json number,title,body,url,state,baseRefName,
// isCrossRepository,mergeCommit,mergedAt` the archive needs.
type PR struct {
	Number            int    `json:"number"`
	Title             string `json:"title"`
	Body              string `json:"body"`
	URL               string `json:"url"`
	State             string `json:"state"`
	BaseRefName       string `json:"baseRefName"`
	IsCrossRepository bool   `json:"isCrossRepository"`
	MergedAt          string `json:"mergedAt"`
	MergeCommit       *struct {
		OID string `json:"oid"`
	} `json:"mergeCommit"`
}

// Manifest is written next to the archived files. InputDigest identifies the
// archive's inputs; a rerun with the same digest is a no-op.
type Manifest struct {
	TaskID      string         `json:"task_id"`
	PR          ManifestPR     `json:"pr"`
	Comment     ManifestSource `json:"records_comment"`
	ListedPaths []string       `json:"listed_paths"`
	Files       []ManifestFile `json:"files"`
	InputDigest string         `json:"input_digest"`
}

type ManifestPR struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	MergeCommit string `json:"merge_commit"`
	MergedAt    string `json:"merged_at"`
}

type ManifestSource struct {
	ID     int64  `json:"id"`
	Author string `json:"author"`
}

// ManifestFile is one archived file. Path is relative to raw/.
type ManifestFile struct {
	Path   string `json:"path"`
	Source string `json:"source"` // "repo" or "inline"
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

// PlanResult tells the shell script what to do next.
type PlanResult struct {
	Status string // "archive" or "skip"
	Reason string
	TaskID string
}

type planInput struct {
	PR       PR
	Comments []Comment
	Repo     string // git work tree the archive is written into
	Tree     string // commit whose files are archived (the merge commit)
	Dest     string // archive root relative to Repo, e.g. docs/changes
	Stage    string // empty directory for the staged archive and prompt
	DiffPath string // optional diff of the PR, for the prompt only
}

// Plan decides whether the PR is archived, copies the listed files verbatim
// from the merge commit into stage/archive/raw/, writes the manifest and the AI
// prompt. It never touches Repo's work tree.
func Plan(in planInput) (PlanResult, error) {
	pr := in.PR
	switch {
	case pr.IsCrossRepository:
		return PlanResult{Status: "skip", Reason: "PR comes from a fork"}, nil
	case pr.State != "MERGED" || pr.MergeCommit == nil || pr.MergeCommit.OID == "":
		return PlanResult{Status: "skip", Reason: fmt.Sprintf("PR is not merged (state %s)", pr.State)}, nil
	case pr.BaseRefName != defaultBaseBranch:
		return PlanResult{Status: "skip", Reason: fmt.Sprintf("PR targets %s, not %s", pr.BaseRefName, defaultBaseBranch)}, nil
	}
	if in.Tree == "" {
		in.Tree = pr.MergeCommit.OID
	}

	recs, comment, found, ignored, err := FindRecords(in.Comments)
	for _, ig := range ignored {
		fmt.Fprintf(os.Stderr, "archive-records: ignoring marker in %s: author is not an owner, member or collaborator\n", ig)
	}
	if err != nil {
		return PlanResult{}, err
	}
	if !found {
		return PlanResult{Status: "skip", Reason: "no records comment"}, nil
	}

	archive := filepath.Join(in.Stage, "archive")
	if err := os.MkdirAll(filepath.Join(archive, "raw"), 0o755); err != nil {
		return PlanResult{}, err
	}

	var files []ManifestFile
	contents := map[string][]byte{}
	total := 0
	add := func(rel, source string, data []byte) error {
		if _, dup := contents[rel]; dup {
			return nil // a file reached through two listed paths is archived once
		}
		if len(data) > maxFileBytes {
			return fmt.Errorf("%s is %d bytes; the limit per archived file is %d", rel, len(data), maxFileBytes)
		}
		if total += len(data); total > maxTotalBytes {
			return fmt.Errorf("archived files exceed %d bytes in total", maxTotalBytes)
		}
		dst := filepath.Join(archive, "raw", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		files = append(files, ManifestFile{Path: rel, Source: source, SHA256: hex.EncodeToString(sum[:]), Bytes: len(data)})
		contents[rel] = data
		return nil
	}

	for _, p := range recs.Paths {
		entries, err := lsTree(in.Repo, in.Tree, p)
		if err != nil {
			return PlanResult{}, err
		}
		for _, e := range entries {
			data, err := git(in.Repo, "cat-file", "blob", e.object)
			if err != nil {
				return PlanResult{}, fmt.Errorf("read %s at %s: %w", e.path, short(in.Tree), err)
			}
			if err := add(e.path, "repo", data); err != nil {
				return PlanResult{}, err
			}
		}
	}
	for _, f := range recs.Inline {
		if err := add("inline/"+f.Name, "inline", []byte(f.Content)); err != nil {
			return PlanResult{}, err
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	m := Manifest{
		TaskID: recs.TaskID,
		PR: ManifestPR{
			Number: pr.Number, Title: pr.Title, URL: pr.URL,
			MergeCommit: pr.MergeCommit.OID, MergedAt: pr.MergedAt,
		},
		Comment:     ManifestSource{ID: comment.ID, Author: comment.User.Login},
		ListedPaths: append(append([]string{}, recs.Paths...), inlinePaths(recs)...),
		Files:       files,
	}
	m.InputDigest = inputDigest(m)

	if existing, err := readManifest(filepath.Join(in.Repo, filepath.FromSlash(in.Dest), recs.TaskID, "manifest.json")); err == nil && existing.InputDigest == m.InputDigest {
		return PlanResult{Status: "skip", Reason: fmt.Sprintf("%s/%s already holds this archive (input digest %s)", in.Dest, recs.TaskID, m.InputDigest[:12]), TaskID: recs.TaskID}, nil
	}

	if err := writeJSON(filepath.Join(archive, "manifest.json"), m); err != nil {
		return PlanResult{}, err
	}
	var diff []byte
	if in.DiffPath != "" {
		if diff, err = os.ReadFile(in.DiffPath); err != nil {
			return PlanResult{}, err
		}
	}
	prompt := buildPrompt(pr, m, contents, diff)
	if err := os.WriteFile(filepath.Join(in.Stage, "prompt.md"), []byte(prompt), 0o644); err != nil {
		return PlanResult{}, err
	}
	return PlanResult{Status: "archive", TaskID: recs.TaskID}, nil
}

func inlinePaths(r Records) []string {
	var out []string
	for _, f := range r.Inline {
		out = append(out, "inline:"+f.Name)
	}
	return out
}

// inputDigest covers what the archive is made of, not when or by whom it was
// produced, so rerunning the same PR yields the same digest.
func inputDigest(m Manifest) string {
	h := sha256.New()
	fmt.Fprintf(h, "task %s\npr %d\n", m.TaskID, m.PR.Number)
	for _, f := range m.Files {
		fmt.Fprintf(h, "%s %s %s\n", f.Source, f.SHA256, f.Path)
	}
	return hex.EncodeToString(h.Sum(nil))
}

type treeEntry struct{ path, object string }

// lsTree lists the regular files at p in tree. A path without a trailing "/"
// must be a file; one with it must be a directory. Symlinks and submodules
// are rejected so the archive only ever holds content stored in the tree.
func lsTree(repo, tree, p string) ([]treeEntry, error) {
	isDir := strings.HasSuffix(p, "/")
	out, err := git(repo, "--literal-pathspecs", "ls-tree", "-r", "-z", "--full-tree", tree, "--", strings.TrimSuffix(p, "/"))
	if err != nil {
		return nil, fmt.Errorf("list %s at %s: %w", p, short(tree), err)
	}
	var entries []treeEntry
	for _, rec := range bytes.Split(out, []byte{0}) {
		if len(rec) == 0 {
			continue
		}
		meta, name, ok := strings.Cut(string(rec), "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			return nil, fmt.Errorf("unexpected ls-tree output %q", rec)
		}
		mode, typ, obj := fields[0], fields[1], fields[2]
		if typ != "blob" || (mode != "100644" && mode != "100755") {
			return nil, fmt.Errorf("%s at %s is a %s (mode %s); only regular files can be archived", name, short(tree), typ, mode)
		}
		entries = append(entries, treeEntry{path: name, object: obj})
	}
	switch {
	case len(entries) == 0:
		return nil, fmt.Errorf("listed path %s does not exist at merge commit %s", p, short(tree))
	case !isDir && (len(entries) != 1 || entries[0].path != p):
		return nil, fmt.Errorf("listed path %s is a directory at %s; write it as %s/", p, short(tree), p)
	case isDir && len(entries) == 1 && entries[0].path == strings.TrimSuffix(p, "/"):
		return nil, fmt.Errorf("listed path %s is a file at %s; drop the trailing /", p, short(tree))
	}
	return entries, nil
}

func git(repo string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func readManifest(p string) (Manifest, error) {
	var m Manifest
	data, err := os.ReadFile(p)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("%s: %w", p, err)
	}
	if m.InputDigest == "" {
		return m, errors.New("manifest has no input_digest")
	}
	return m, nil
}

func writeJSON(p string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644)
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
