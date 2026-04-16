# Final Implementation v1

EBTA-005 will add Kimi native `UsageEvent` support in `provider/kimi` only.

Required behavior:

- `Provider.CollectUsageEvents` scans local Kimi sessions from the explicit base directory, or the same default sessions directory as `CollectSessions`.
- Each `StatusUpdate` with `token_usage` becomes one event.
- Event timestamps come from the `StatusUpdate` line.
- Event token usage is incremental and equals the payload values.
- Event metadata preserves `ProviderName`, `SessionID`, `Title`, `WorkDirHash`, and `ModelName` behavior from session parsing.
- Metadata model fields win over payload model fields; payload model fields win over log fallback.
- Existing Kimi session tests continue to pass.

Review result:

- Explorer A and Explorer B found no high-severity design issue.
- Key risk recorded: future real Kimi logs may prove `token_usage` is cumulative, but this task intentionally keeps the current incremental assumption.
