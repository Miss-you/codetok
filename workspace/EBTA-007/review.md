# EBTA-007 Artifact Review

## Review 1

Score: 77/100, fail.

Findings:

- Verification gates were too weak because `go test ./cmd -run TestDaily` could pass without exercising EBTA-007 behavior.
- The command-level test source was underspecified.
- Date-window validation order could allow provider collection errors to mask flag errors.
- OpenSpec artifacts were incomplete at the time of review.

Actions:

- Strengthened test names and verification commands.
- Added validation-order requirement and test.
- Completed `event-based-token-aggregation-daily` OpenSpec artifacts.

## Review 2

Score: 93/100, pass.

Remaining non-blocking clarifications:

- Command-level tests should use a deterministic fake event provider with a unique provider filter.
- Default `--days` local-midnight behavior should use fixed-time helper coverage instead of wall-clock `runDaily`.

Actions:

- Added those constraints to `test_strategy.md` and `final_impl.md`.

## Code Review

Result: pass, no must-fix issues.

Reviewer notes:

- `daily` validates the date window before collection.
- `daily` collects usage events, filters by localized date keys, and delegates grouping to the existing event stats helper.
- `session` command code was not changed.
- Cursor/local-only behavior still goes through the existing local collection bridge.

Residual risk:

- Claude subagent daily `sessions` count now follows the event aggregation contract: distinct contributing session IDs. Parent and subagent files sharing `session-main` report `1`, while `session` still reports two session rows.

## PR AI Review

Copilot left two test-quality comments:

- Use a unique provider name in the validation-order fake provider to avoid global registry collisions.
- Keep fake `UsageEvent.ProviderName` and `SessionInfo.ProviderName` aligned with the selected fake provider name.

Actions:

- Added `newDailyEventTestProviderName`.
- Updated `registerDailyEventTestProvider` to stamp events and sessions with the generated provider name.
- Updated the validation-order test to use a generated provider name.
