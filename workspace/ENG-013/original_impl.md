# ENG-013 original implementation

State of `origin/main` before this task (2026-11-20):

- No scheduled automation. `.github/workflows/` has `ci.yml` (lint, test, build matrix on GitHub-hosted runners) and `release.yml`.
- `scripts/ai-review.sh` (#54/#55) runs a read-only review of a git range with `acpx` + `agy` (`gemini-4-hydrogen`) and prints a review in the superpowers `code-reviewer.md` format ("Ready to merge? Yes/No/With fixes", Critical/Important/Minor sections). It is run by hand after each PR push.
- Cleanup work (lint backlog, dead code, doc drift, test gaps) is found and done by hand. Today `golangci-lint` is clean, `deadcode -test` finds nothing, but `deadcode` without `-test` reports 10 functions only tests reach (e.g. `cmd.collectUsageEvents`, `stats.AggregateByDay`), `README*.md` never mention `cursor login --token`, and every tested package is above 70% coverage.
- The self-hosted runner (#56, ENG-011) is being set up in parallel; its labels are `self-hosted, codetok-local, ai`.
