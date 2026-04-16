# EBTA-009 Original Implementation

`daily` and `session` are already event-based on the current `main` baseline:

- `daily` collects usage events, filters them by localized event date, and aggregates rows by date plus grouping dimension.
- `session` collects the same usage events, filters them by localized event date, and groups only the included events by provider/session.
- Existing e2e tests cover Kimi, Claude subagents, Cursor, and generic JSON output, but they do not prove a single provider session that crosses midnight is split by event date at the binary level.

The current e2e fixtures are provider-specific roots:

- Kimi: `e2e/testdata/sessions`
- Claude: `e2e/testdata/claude-sessions`
- Cursor: `e2e/testdata/cursor`

Adding cross-day fixtures to those roots would perturb existing tests that assert exact row counts and token totals.
