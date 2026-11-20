## Context

Antigravity stores one SQLite file per conversation (`conversations/<cascade_id>.db`). Blobs are protobuf without a published schema. Field names and numbers below were read from the descriptors embedded in the `agy` binary (agy 1.5.13) and checked against 300 local conversation files (3421 generation rows).

### Tables used

- `gen_metadata(idx, data)`: one row per model call. `data` is `CortexStepGeneratorMetadata`.
- `steps(idx, metadata, ...)`: one row per trajectory step. `metadata` is `CortexStepMetadata`.

### Fields used

| Message | Field | Name | Use |
|---|---|---|---|
| `CortexStepGeneratorMetadata` | 1 | `chat_model` (`ChatModelMetadata`) | usage and model |
| `CortexStepGeneratorMetadata` | 2 | `step_indices` (packed uint32) | first step gives the timestamp |
| `ChatModelMetadata` | 4 | `usage` (`ModelUsageStats`) | token counts |
| `ChatModelMetadata` | 19 | `response_model` (string) | model name |
| `ChatModelMetadata` | 21 | `model_display_name` (string) | model fallback |
| `ModelUsageStats` | 2 | `input_tokens` | `InputOther` |
| `ModelUsageStats` | 3 | `output_tokens` | `Output` (already includes thinking: `output = thinking_output(9) + response_output(10)` in every sampled row) |
| `ModelUsageStats` | 4 | `cache_write_tokens` | `InputCacheCreate` |
| `ModelUsageStats` | 5 | `cache_read_tokens` | `InputCacheRead` (`input_tokens` excludes it: a call's cache read equals the previous call's prompt) |
| `CortexStepMetadata` | 1 | `created_at` (Timestamp) | event time |
| `CortexStepMetadata` | 7 | `finished_generating_at` | event time fallback |

`ChatModelMetadata` fields 11/12 are durations (`time_to_first_token`, `streaming_duration`), not timestamps.

## Decisions

- **Unit of counting is the `gen_metadata` row.** Every step also carries `model_usage` (field 9), but one call can produce several steps with the same usage, so summing steps over-counts. In all 3377 sampled rows with step indices, the row's usage equals its first step's usage.
- **Rows without usage or with all-zero usage are skipped.** The 44 sampled rows without step indices all had zero usage.
- **Timestamp** comes from the first step's `created_at`, falling back to `finished_generating_at`. A row with no timestamp is skipped.
- **Opening files:** `mode=ro` fails on these files, so the provider opens them with `immutable=1` (no locks, no writes). `immutable` ignores the WAL, so when `<db>-wal` is non-empty the provider copies the DB and WAL to a temp directory and opens the copy, which keeps the latest calls of a running conversation. The two files are copied separately, so the copy is used only when the size and mtime of both files are unchanged after copying; otherwise it retries (3 attempts), since a checkpoint in between would pair a WAL with the wrong database. Paths are made absolute before building the `file:` URI.
- **Directories:** default roots are `~/.gemini/antigravity-cli` and `~/.gemini/antigravity`; missing roots are skipped. `--antigravity-dir` (or `--base-dir`) replaces both with one root that contains `conversations/`.
- **Session metadata:** session ID is the file name (cascade ID). Title comes from `conversation_summaries.db` in the same root when available. Session model is the most frequent call model. Turns is the number of counted calls.
- **Range pruning:** files whose mtime is before `Since` are skipped, like other providers.

## Risks

- The schema is private and can change. Unknown fields are ignored; a row that cannot be decoded is skipped instead of failing the run.
- Subagent conversations are separate files; no duplicate usage across files was found in the sample, so no cross-file dedup is added.
