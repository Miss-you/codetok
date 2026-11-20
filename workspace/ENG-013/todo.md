# ENG-013 remaining work

Before `done` (needs the maintainer, outward-facing):

- [ ] Create the label: `gh label create auto-optimize -R Miss-you/codetok --color 0E8A16 --description "Automated cleanup PR (scripts/auto-optimize.sh, #58)"`. Without it, PR mode stops with an error before doing anything.
- [ ] After ENG-011 is merged and the runner with labels `codetok-local`, `ai` is online: run `gh workflow run auto-optimize.yml -R Miss-you/codetok -f mode=dry-run`, then `-f mode=pr`. Check the PR is labeled, within 200 lines, has an `AI review:` comment, and CI ran. Then run `mode=pr` again and check it skips because one PR is open.
- [ ] Confirm the runner's launchd environment (ENG-011 conventions) has `go`, `golangci-lint`, `gh`, `jq`, `acpx`, `agy` on `PATH`, a git identity and credential helper that can push, and `AUTO_OPTIMIZE_PROXY` / `HTTPS_PROXY` for agy.
- [ ] Decide whether to enable the weekly schedule: `gh variable set AUTO_OPTIMIZE_SCHEDULE -R Miss-you/codetok --body true`.

Later (not in this task):

- Optional post-merge trigger (the issue lists it as optional; only schedule and manual for now).
- If merge rate stays low for a candidate kind, drop or retune that kind (`AUTO_OPTIMIZE_MIN_COVERAGE`, ignore lists in `scripts/autooptimize/candidates.go`).
