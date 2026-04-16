## Context

The shared `provider.UsageEvent` model already exists and represents timestamped token deltas. Kimi's current parser reads `baseDir/<work-dir-hash>/<session-id>/wire.jsonl`, parses `metadata.json`, and sums every `StatusUpdate.token_usage` into a single `SessionInfo.TokenUsage`.

The EBTA-005 task is provider-scoped. It should not switch command behavior, alter date filtering, or infer cumulative Kimi semantics that are not proven by current fixtures.

## Decisions

- Add `CollectUsageEvents(baseDir string)` to the Kimi provider and keep `CollectSessions` unchanged.
- Reuse the existing Kimi session discovery shape and log model fallback.
- Add a `parseUsageEvents(sessionPath, workDirHash, sessionModelIndex)` helper that reads metadata once, then scans `wire.jsonl`.
- Emit one event for each valid `StatusUpdate` that includes a `token_usage` object, using that line's timestamp.
- Treat `StatusUpdate.token_usage` as incremental, matching existing session tests and the task plan.
- Populate event metadata from the same priority order as sessions:
  metadata model fields, then `StatusUpdate` model fields, then log fallback.
- Set `SourcePath` to the wire file path and build `EventID` from source path plus line number/message ID where available for traceability.

## Risks

- Real Kimi logs may report cumulative values in some versions. This task intentionally preserves the current incremental assumption; cumulative detection should be a later task backed by real fixtures.
- `provider.ParseParallel` currently returns `[]SessionInfo`, so native event collection either needs sequential parsing or a local goroutine helper. For the small provider-scoped task, keep the implementation simple unless tests show performance concerns.

## Validation

- `go test ./provider/kimi -run 'Test(ParseKimiUsageEvents|CollectKimiUsageEvents)'` must fail before implementation and pass after.
- `go test ./provider/kimi` must keep existing session behavior green.
- Repository gates still run before closing the task.
