# EBTA-009 Final Implementation V1

Implement EBTA-009 as e2e-only acceptance coverage.

Files to touch:

- `e2e/e2e_test.go`
- `e2e/testdata/cross-day/**`
- `docs/plans/2026-04-16-event-based-token-aggregation-task.md`
- `workspace/EBTA-009/**`

Fixture expectations:

| Provider | 2026-04-15 Total | 2026-04-16 Total | 2026-04-16 Session Total |
| --- | ---: | ---: | ---: |
| codex | 1300 | 650 | 650 |
| claude | 16 | 35 | 35 |
| kimi | 360 | 545 | 545 |

Implementation notes:

- Codex uses cumulative `total_token_usage`; the second event must assert the delta only.
- Claude uses assistant `message.usage` timestamps.
- Kimi uses incremental `StatusUpdate.token_usage` Unix timestamps.
- Keep provider roots isolated from existing e2e fixtures to avoid changing unrelated tests.
- Index JSON rows by provider/date or provider/session rather than relying on row order.
