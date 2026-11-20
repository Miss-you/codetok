# ENG-012 Verification

2026-11-20 CST, worktree `claude/eng-012-post-merge-archive-2e87cd`, macOS arm64, Go 1.26.0.

| Command | Result |
| --- | --- |
| `make fmt` | clean |
| `make vet` | pass |
| `make test` | pass (rerun after AI review round 1 fixes); all packages pass |
| `make build` | pass |
| `make lint` (golangci-lint) | 0 issues |
| `go test ./scripts/archive-records/ -run TestArchiveScript -v` | 5/5 subtests pass (archive, rerun no-op, no-marker skip, fork skip, invalid AI output) |
| acpx + agy smoke (`--deny-all --no-terminal`, tiny prompt) | exit 0 in 12 s; output contains the `BEGIN_ARCHIVE_SUMMARY` block followed by `[done] end_turn`, which `ExtractSummary` handles |

Not run: shellcheck (not installed); the real workflow on the runner (see todo.md).
