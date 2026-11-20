# ENG-011 Original Implementation

State on `origin/main` at 2a4d0e9 (2026-11-20):

- CI: `.github/workflows/ci.yml` (lint, test matrix, build matrix) and `release.yml` run only on GitHub-hosted runners (`ubuntu-latest`, `macos-latest`).
- No self-hosted runner is registered: `gh api repos/Miss-you/codetok/actions/runners` → `total_count: 0`.
- AI review exists only as a local script, `scripts/ai-review.sh` (#54), which drives `acpx` + `agy acp` and needs the local proxy (`AI_REVIEW_PROXY`, default `http://127.0.0.1:7890`). #54 explicitly scoped "Running the review in GitHub Actions" out because hosted runners have neither the tools nor the login state.
- Repository Actions settings: Actions enabled, all actions allowed, default `GITHUB_TOKEN` permissions `read`, fork-PR approval policy `first_time_contributors`.
- This Mac: Apple Silicon Mac mini, macOS 26.4.1, FileVault on, no auto-login. Tools: `go`, `gh`, `acpx`, `node`, `python3` in `/opt/homebrew/bin`; `agy` in `~/.local/bin`; `golangci-lint` in `~/go/bin`. `agy -p` works under `env -i` (minimal env) as long as `HOME` and `HTTPS_PROXY` are set.
