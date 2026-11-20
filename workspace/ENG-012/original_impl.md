# ENG-012 Original Implementation

State of `main` at 2a4d0e9 (2026-11-20):

- Process files live in three places and are never collected: `openspec/changes/<change>/`, `docs/plans/*.md` task boards, and `workspace/<task-id>/` (EBTA-*, EAP-*, ENG-*). Nothing links them to the PR that delivered them.
- `delivering-go-task-end-to-end` ends at "close the task"; its handoff only points to `monitoring-pr-ai-reviews`. There is no step that hands records to anything after the merge.
- AI review (#54 / PR #55) is `scripts/ai-review.sh --pr N`, run locally by the agent after every push to a PR head. It is not a workflow. Any commit added to the PR head (for example an archive commit) would need another review round.
- `.github/workflows/` has `ci.yml` (push to main + pull_request) and `release.yml` (tags). No self-hosted job exists yet; ENG-011 (#56) registers the runner `codetok-mac-mini` with labels `self-hosted, macOS, ARM64, codetok-local, ai`.
- The runner user has `gh` logged in and `gh auth git-credential` configured as the git credential helper for github.com, so `git push` over https works without repo secrets.
- `openspec/changes/archive/` does not exist yet; the `openspec-archive-change` skill moves a change directory there and syncs specs.
