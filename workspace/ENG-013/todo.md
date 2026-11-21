# ENG-013 remaining work

Before `done` (needs the maintainer, outward-facing):

- [x] Create the label (done 2026-11-21): `gh label create auto-optimize -R Miss-you/codetok --color 0E8A16 --description "Automated cleanup PR (scripts/auto-optimize.sh, #58)"`. Without it, PR mode stops with an error before doing anything.
- [ ] ENG-011 is merged and three runners with labels `codetok-local`, `ai` are online (2026-11-21). `workflow_dispatch` needs the workflow file on `main`, so after this PR is merged run `gh workflow run auto-optimize.yml -R Miss-you/codetok -f mode=dry-run`, then `-f mode=pr`. Check the PR is labeled, within 200 lines, has an `AI review:` comment, and CI ran. Then run `mode=pr` again and check it skips because one PR is open.
- [x] Runner environment: the workflow pins `/opt/homebrew/bin`, `~/go/bin`, `~/.local/bin` (ENG-011 convention) and passes `AI_REVIEW_PROXY`; this Mac's git uses `gh auth git-credential` for github.com and has a git identity. The dry run confirms the rest.
- [ ] Decide whether to enable the weekly schedule: `gh variable set AUTO_OPTIMIZE_SCHEDULE -R Miss-you/codetok --body true`.
- [ ] After enabling the schedule, check that the first scheduled run is not skipped by the owner-only guard (`github.triggering_actor == github.repository_owner`): `gh run list -R Miss-you/codetok --workflow auto-optimize.yml --event schedule --limit 1 --json conclusion,url`. A `skipped` conclusion means the cron line was last changed by someone other than the owner.

Later (not in this task):

- A full-pipeline test of `scripts/auto-optimize.sh` with fake `acpx`/`gh`/`make` on `PATH`. The safety helpers are already unit-tested from Go (`shell_test.go`); the end-to-end flow is covered by the local rehearsals and the runner check above.

- Optional post-merge trigger (the issue lists it as optional; only schedule and manual for now).
- If merge rate stays low for a candidate kind, drop or retune that kind (`AUTO_OPTIMIZE_MIN_COVERAGE`, ignore lists in `scripts/autooptimize/candidates.go`).
