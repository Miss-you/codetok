# Auto-optimize: scheduled small cleanup PRs

`.github/workflows/auto-optimize.yml` runs `scripts/auto-optimize.sh` on the self-hosted runner on this Mac (`runs-on: [self-hosted, codetok-local, ai]`, see #56). Each run proposes **at most one** small cleanup PR for a person to review. It never merges anything. Issue: #58.

## Trigger

- **Manual**: Actions → *Auto optimize* → *Run workflow*, or
  `gh workflow run auto-optimize.yml -R Miss-you/codetok -f mode=pr`.
  `mode` is `pr` (default), `no-pr` (everything except push and PR; the commit stays on a local `auto-optimize/*` branch on the runner) or `dry-run` (only list candidates and pick one).
- **Scheduled**: Mondays 02:00 UTC (10:00 Beijing). Scheduled runs are skipped unless the repository variable `AUTO_OPTIMIZE_SCHEDULE` is `true`:
  `gh variable set AUTO_OPTIMIZE_SCHEDULE -R Miss-you/codetok --body true`.
- **Never** on pull requests (there is no `pull_request` trigger, so fork PRs cannot start it) and never outside `Miss-you/codetok`.
- Locally: `scripts/auto-optimize.sh --dry-run` on any checkout is read-only. `--no-pr` and the default mode check out `origin/main` and create a branch, so run them only on a clean, disposable checkout.

## What one run does

1. **Open-PR limit.** If an open PR with the `auto-optimize` label exists, stop. At most one unmerged optimization PR at a time.
2. **Candidates from fixed tools, no AI** (on `origin/main`):
   - `golangci-lint` findings (`.golangci.yml`);
   - `deadcode -json ./...` (golang.org/x/tools, pinned): functions nothing reachable from `main` calls, that is, code only tests keep alive;
   - `README.md` / `README_zh.md` flag drift: `--flags` the READMEs mention that `cmd/` does not define, and flags `cmd/` defines that a README never mentions;
   - packages that have tests but statement coverage below 60% (`go test -cover`, `e2e` left out).

   Paths that are never candidates and never changed: `.github/`, `scripts/`, `docs/`, `npm/`, `workspace/`, `openspec/`, `skills/`, agent config dirs, `go.mod`, `go.sum`, `Makefile`, `AGENTS.md`, `CLAUDE.md`, lint and release config.
3. **Pick one.** Candidates are sorted lint → deadcode → docs → coverage (lowest coverage first), then by file and line. The first one is taken, skipping candidates whose earlier PR was closed without merging and candidates an earlier run abandoned (`~/.local/state/codetok-auto-optimize/abandoned.txt` on the runner).
4. **Change.** The change agent (`acpx` + `agy`, model `claude-sonnet-5-5`) gets read and edit permissions only; shell commands and URL fetches are denied, and its transcript leaves out file contents (`--suppress-reads`). Before each agent turn the script snapshots git's config, hooks and excludes, the ignored files in the checkout, and its own state (tool binary, chosen candidate, round counter); if the agent changed any of them, the script restores git's files, deletes the new ignored files and ends the run without a PR. Allowed files:
   - lint, deadcode: `.go` files directly in the candidate's package directory, and its `testdata/`;
   - coverage: only `*_test.go` in that directory, and its `testdata/`;
   - docs: only `README.md` and `README_zh.md`.
5. **Checks.** The script stages the change, runs `gofmt` on changed Go files, then checks scope and size (≤ 200 added+deleted lines). Only a change inside its scope is built: `make test` and `make lint` then run with a scrubbed environment (empty `HOME`, no proxy or tokens, `GOPROXY=off`). On failure the agent gets the errors and tries again, **at most 2 fix rounds**. Still failing, a rejected review, or no change at all: the run ends without a PR and records the candidate as abandoned. A tool or service failure (agent timeout, proxy down, reviewer error) also ends the run without a PR but does not record the candidate, so the next run tries it again.
6. **Independent review.** `scripts/ai-review.sh` reviews the commit with a different model (`gemini-4-hydrogen`) against the candidate and the "no behavior change, no scope creep" rules. The run continues only when the verdict is not "No" and the Critical and Important sections are empty.
7. **PR.** Push `auto-optimize/<date>-<candidate id>`, open a PR with the `auto-optimize` label and the candidate, change, checks and review in the body, and post the review as an `AI review: <head SHA>` comment. CI runs as for any PR. A person reviews and merges, or closes it.

Every decision in steps 1, 3, 5 and 6 (and the candidate list in step 2) is made by `scripts/autooptimize` (Go, tested by `make test`; `go test ./scripts/autooptimize` alone). A run that ends without a PR still exits 0 and writes the reason to the log and the job summary. Reports and check output (not the agent or reviewer transcripts) are uploaded as the `auto-optimize-<run id>` artifact for 14 days; everything else stays in `$RUNNER_TEMP/auto-optimize` on the runner.

## Credentials and trust

Nothing secret is stored in the repository or the workflow. The job's `GITHUB_TOKEN` is read-only. `git push`, `gh` and `agy` use the runner machine's existing logins (git credential helper, `gh auth`, the Antigravity CLI login), so the PR is authored by the machine's GitHub user and triggers CI. `agy` needs a proxy in a supported region: set `AUTO_OPTIMIZE_PROXY` (default: `AI_REVIEW_PROXY`, `HTTPS_PROXY`, then `http://127.0.0.1:7890`) in the runner's environment.

Limits to be aware of:

- `make test` runs code the agent wrote (tests, and package code in lint/deadcode runs). The scrubbed environment keeps honest test code away from the logins, but it is not a sandbox: the process still runs as the runner's user. The agent only sees code from `main`, never fork or PR content, and every PR is reviewed by a person before merging.
- The agent can read any file the runner user can read. Its transcript omits file contents, the PR body no longer quotes the agent, the review comment carries only the reviewer's answer (no tool trace), and the PR body and comment are scanned for token-like strings before anything is published; a hit ends the run without publishing.

## Settings

Environment variables read by `scripts/auto-optimize.sh` (set them in the runner's environment or the workflow's `env:`):

| Variable | Default | Meaning |
| --- | --- | --- |
| `AUTO_OPTIMIZE_LABEL` | `auto-optimize` | PR label; must exist |
| `AUTO_OPTIMIZE_MAX_OPEN` | `1` | open optimization PRs allowed |
| `AUTO_OPTIMIZE_MAX_LINES` | `200` | diff cap (added + deleted lines) |
| `AUTO_OPTIMIZE_MAX_ROUNDS` | `2` | fix rounds after the first attempt |
| `AUTO_OPTIMIZE_MIN_COVERAGE` | `60` | coverage threshold for candidates |
| `AUTO_OPTIMIZE_MODEL` | `claude-sonnet-5-5` | change agent model (reviewer: `AI_REVIEW_MODEL`) |
| `AUTO_OPTIMIZE_TIMEOUT` | `1800` | seconds per agent turn |
| `AUTO_OPTIMIZE_STATE_DIR` | `~/.local/state/codetok-auto-optimize` | keeps `abandoned.txt` |
| `AUTO_OPTIMIZE_OUT_DIR` | `$TMPDIR/codetok-auto-optimize/<time>` | logs and reports, outside the checkout |
| `DEADCODE_VERSION` | `v0.52.0` | `golang.org/x/tools` version for `deadcode` |

To retry an abandoned candidate, delete its line from `abandoned.txt`. A candidate whose PR was closed unmerged is skipped for good; its ID is in the PR body (`<!-- auto-optimize-candidate: … -->`).

## Turning it off

- Stop scheduled runs: `gh variable delete AUTO_OPTIMIZE_SCHEDULE -R Miss-you/codetok` (manual runs still work).
- Stop everything: `gh workflow disable auto-optimize.yml -R Miss-you/codetok` (undo with `gh workflow enable`).

## Merge rate

All optimization PRs carry the `auto-optimize` label. Counts by state:

```bash
gh pr list -R Miss-you/codetok --label auto-optimize --state all --limit 1000 --json state \
  --jq 'group_by(.state) | map({key: .[0].state, value: length}) | from_entries'
```

Merge rate = `MERGED / (MERGED + CLOSED)`; open PRs are still undecided:

```bash
gh pr list -R Miss-you/codetok --label auto-optimize --state all --limit 1000 --json state \
  --jq '[.[] | select(.state != "OPEN")] as $d | if ($d | length) == 0 then "no decided PRs yet"
        else "\([$d[] | select(.state == "MERGED")] | length)/\($d | length) merged" end'
```

To see why PRs were rejected, read the closed ones: `gh pr list -R Miss-you/codetok --label auto-optimize --state closed --json number,title,url,mergedAt --jq '.[] | select(.mergedAt == null)'`.
