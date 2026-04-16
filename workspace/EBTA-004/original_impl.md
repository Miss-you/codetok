# EBTA-004 Original Claude Implementation Notes

Scope: current Claude session collection in `provider/claude/parser.go`, tests in
`provider/claude/parser_test.go`, and shared event definitions in
`provider/provider.go`.

## Session discovery and parsing

- `(*Provider).CollectSessions` is the only Claude collection entry point today.
  With an explicit `baseDir`, it scans only that directory. With an empty
  `baseDir`, it scans `~/.claude/projects` and `~/.claude-internal/projects`.
  Missing default directories are ignored; an explicit missing directory returns
  an error. See `provider/claude/parser.go:55`.
- Claude does not currently implement `provider.UsageEventProvider`; the shared
  optional event interface exists in `provider/provider.go:68`.
- `collectPaths` expects project directories under the base directory. It records
  top-level `*.jsonl` files at `baseDir/<project-slug>/*.jsonl` and subagent
  files at `baseDir/<project-slug>/<session-dir>/subagents/*.jsonl`. It stores
  the project directory name in `pathToSlug[path]`. See
  `provider/claude/parser.go:99`.
- `CollectSessions` parses the collected paths with `provider.ParseParallel`,
  passing each path to `parseSession(path, pathToSlug[path])`. See
  `provider/claude/parser.go:91`.
- `parseSession` scans one JSONL file with a 1 MiB scanner buffer, skips empty
  lines, skips malformed JSON lines, and parses timestamps with
  `time.RFC3339Nano`. See `provider/claude/parser.go:150`.
- The parsed Claude event fields currently used are `type`, `userType`,
  `sessionId`, `requestId`, `timestamp`, `message.id`, `message.model`,
  `message.role`, `message.content`, and `message.usage`. See
  `provider/claude/parser.go:28`.

## Current SessionInfo aggregation

- `parseSession` initializes `SessionInfo.ProviderName` to `"claude"` and
  `SessionInfo.WorkDirHash` to the project slug passed from discovery. See
  `provider/claude/parser.go:158`.
- Every valid timestamp can update `StartTime` and `EndTime`; the parser keeps
  the minimum and maximum timestamps seen in the file. See
  `provider/claude/parser.go:195`.
- `type:"user"` events increment `Turns` unless `userType` is a non-empty value
  other than `"external"`. The first accepted user message becomes the session
  title via `extractUserText`, then is truncated to 80 runes. See
  `provider/claude/parser.go:206` and `provider/claude/parser.go:262`.
- `type:"assistant"` events set `SessionID` from `event.sessionId` if empty and
  set `ModelName` from the first non-empty `message.model`. See
  `provider/claude/parser.go:217`.
- Assistant `message.usage` is aggregated only after deduplication. The parser
  maps Claude `input_tokens` to `TokenUsage.InputOther`,
  `cache_read_input_tokens` to `InputCacheRead`,
  `cache_creation_input_tokens` to `InputCacheCreate`, and `output_tokens` to
  `Output`. The deduplicated entries are summed into `SessionInfo.TokenUsage`
  after scanning the file. See `provider/claude/parser.go:226` and
  `provider/claude/parser.go:247`.
- If no event supplies a session ID, the filename without `.jsonl` is used. See
  `provider/claude/parser.go:256`.
- Tests cover normal aggregation, malformed-line skipping, empty files, sessions
  without assistant usage, explicit directory behavior, subagent discovery, and
  multiple projects. See `provider/claude/parser_test.go:9`,
  `provider/claude/parser_test.go:215`, and
  `provider/claude/parser_test.go:265`.

## Streaming duplicate handling

- Streaming duplicates are handled inside `parseSession` by `dedupUsage`, keyed
  by `message.id + ":" + requestId`. A later row with the same key overwrites
  the prior usage entry, so only the last row in file order is counted. See
  `provider/claude/parser.go:167` and `provider/claude/parser.go:271`.
- If both `message.id` and `requestId` are empty, `dedupKey` returns a generated
  unique key, so those rows are never merged. Partial IDs still dedup by the
  non-empty side, producing keys like `msg-X:` or `:req-Y`. See
  `provider/claude/parser.go:273`.
- Tests explicitly assert last-row-wins streaming behavior, no dedup when both
  IDs are absent, and partial-ID behavior. See
  `provider/claude/parser_test.go:76`, `provider/claude/parser_test.go:129`,
  and `provider/claude/parser_test.go:154`.

## Fields available for native usage events

- Shared `UsageEvent` fields are `ProviderName`, `ModelName`, `SessionID`,
  `Title`, `WorkDirHash`, `Timestamp`, `TokenUsage`, `SourcePath`, and
  `EventID`. See `provider/provider.go:36`.
- `ProviderName`: currently constant `"claude"` from `Provider.Name` and
  `parseSession` initialization. See `provider/claude/parser.go:23` and
  `provider/claude/parser.go:158`.
- `SessionID`: available from each parsed Claude event as `sessionId`; current
  session fallback is the JSONL filename stem. See `provider/claude/parser.go:32`
  and `provider/claude/parser.go:256`.
- `ModelName`: available on assistant messages as `message.model`; session
  aggregation keeps the first non-empty model only. See
  `provider/claude/parser.go:40` and `provider/claude/parser.go:221`.
- `WorkDirHash`: available from discovery as the project directory slug, not
  from the JSONL `cwd` field. The current `claudeEvent` struct does not parse
  `cwd`. See `provider/claude/parser.go:76` and
  `provider/claude/parser.go:158`.
- `Title`: derived from the first accepted user message content. Assistant usage
  rows do not carry their own title. See `provider/claude/parser.go:212` and
  `provider/claude/parser.go:281`.
- `Timestamp`: available from every parsed event as a string and currently
  parsed with `time.RFC3339Nano`. For usage events, the assistant usage row
  timestamp is available. See `provider/claude/parser.go:34` and
  `provider/claude/parser.go:195`.
- `SourcePath`: available to any native event collector because discovery has
  the JSONL path. It is not present in `SessionInfo` and is not currently
  retained by `parseSession`. See `provider/provider.go:45` and
  `provider/claude/parser.go:150`.

## Constraints and risks for native CollectUsageEvents

- `CollectSessions` behavior must remain unchanged. Existing tests rely on
  current discovery, explicit-vs-default directory errors, title extraction,
  turn counting, session ID fallback, first-model selection, and session-level
  deduplication.
- Native `CollectUsageEvents` should reuse the same file discovery semantics as
  `CollectSessions`, including subagent discovery and project slug mapping, or
  the event path will disagree with current session totals.
- The event collector must preserve the same streaming dedup semantics before
  emitting events. Emitting every assistant usage row would double-count partial
  streaming rows that `CollectSessions` currently collapses.
- Current dedup is "last row in file order wins", not "largest timestamp wins".
  Refactors should not silently change that unless tests and session behavior are
  updated together.
- If a shared parser is introduced, it needs to carry per-entry metadata that
  `SessionInfo` does not need today: timestamp, source path, and ideally a stable
  event ID. Current `claudeEvent` does not parse Claude's `uuid`, even though
  fixtures contain it, so `EventID` would need either a struct extension or a
  derived value such as source path plus dedup key.
- Usage rows should be treated as final assistant-message token deltas after
  deduplication. They should not be converted into incremental deltas by
  subtracting earlier streaming rows, because session aggregation currently keeps
  only the final usage counts.
- Keep reporting local-only. Adding `CollectUsageEvents` for Claude should read
  existing JSONL files only and must not add any implicit Claude remote API work.
