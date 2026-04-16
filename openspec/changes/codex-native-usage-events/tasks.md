## 1. Tests

- [x] 1.1 Add failing Codex event parser tests for `last_token_usage`, cumulative deltas across dates, cumulative reset handling, stable first session/title metadata, model context fallback, `CODEX_HOME`, and explicit directory precedence.

## 2. Implementation

- [x] 2.1 Refactor Codex source directory resolution and JSONL file discovery into shared helpers.
- [x] 2.2 Implement `Provider.CollectUsageEvents` with bounded parallel file parsing.
- [x] 2.3 Implement `parseCodexUsageEvents` with last-usage precedence, cumulative delta recovery, reset handling, and metadata preservation.

## 3. Validation

- [x] 3.1 Run focused Codex event tests.
- [x] 3.2 Run `go test ./provider/codex`.
- [x] 3.3 Run repository gates required by the EBTA workflow before closing the task.
