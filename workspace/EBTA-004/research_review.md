# EBTA-004 Research Review

Multi-agent review found no high-severity issue with the final implementation direction.

Adopted review points:

- Reuse existing same-package test style in `provider/claude/parser_test.go`.
- Reuse `collectPaths` so parent and subagent discovery match `CollectSessions`.
- Keep no-ID assistant usage records unique.
- Skip usage events with invalid timestamps to avoid `0001-01-01` aggregation.
- Preserve session parser behavior and tests unchanged.
- Avoid forcing `provider.ParseParallel` into native event collection because it is session-specific.

Rejected or deferred points:

- Adding a shared provider-level event parallel parser is out of scope for EBTA-004 and can be considered after more providers implement native events.
- Parsing Claude top-level `uuid` for `EventID` is not required for this task; the existing message/request dedupe key is stable for the task's streaming contract.
