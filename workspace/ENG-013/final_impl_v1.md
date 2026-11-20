# ENG-013 final implementation v1

As in `new_impl.md`, with these rules fixed:

- Candidate order: lint → deadcode → docs → coverage (coverage lowest first), then file, line, ID. ID = first 12 hex of sha256(kind, package, file, source, message); line numbers excluded so unrelated edits do not create "new" candidates.
- deadcode runs without `-test` (production-unreachable code); `internal/testutil` and generated functions are skipped.
- Coverage candidates only for packages that already have tests (first test suites are design work).
- Never candidates / never in a diff: `.github/ .claude/ .codex/ .trae/ scripts/ npm/ docs/ workspace/ openspec/ skills/ vendor/`, `go.mod go.sum Makefile AGENTS.md CLAUDE.md .golangci.yml .goreleaser.yml .gitignore`.
- Scope: lint/deadcode → `.go` directly in the package dir + its `testdata/`; coverage → only `*_test.go` + `testdata/`; docs → only `README.md`, `README_zh.md`. Cap 200 added+deleted lines, no binaries, no empty diff.
- Fix budget: one first attempt plus at most 2 fix rounds; scope violations count as failures too.
- Review gate: pass iff verdict is not "No" and Critical + Important sections hold no issues (prose without bullets counts as one issue; "None" does not).
- Rejection memory: candidates of PRs closed unmerged are skipped (marker `<!-- auto-optimize-candidate: ID -->` in the PR body); abandoned candidates go to `~/.local/state/codetok-auto-optimize/abandoned.txt` on the runner.
