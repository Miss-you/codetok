# EBTA-009 Candidate Implementation

Add a dedicated fixture root under `e2e/testdata/cross-day/`:

- `codex/2026/04/15/rollout-2026-04-15T23-50-00-cross-day.jsonl`
- `claude/project-cross-day/cross-day.jsonl`
- `kimi/hash-cross-day/session-cross-day/{metadata.json,wire.jsonl}`

Add e2e helpers that resolve these roots and isolate Cursor with a temp directory.

Add one focused e2e test that builds the binary once and runs two JSON checks:

1. `daily --json --since 2026-04-15 --until 2026-04-16 --timezone UTC`
   - expects one row per provider per date
   - checks `group_by=cli`, provider group names, distinct session count, and exact token usage
2. `session --json --since 2026-04-16 --until 2026-04-16 --timezone UTC`
   - expects one row per provider
   - checks only the 2026-04-16 event deltas are included
   - checks `date=2026-04-16` and one included turn/event per session

No new OpenSpec delta is needed because this task adds acceptance coverage for already specified behavior; it does not change product behavior.
