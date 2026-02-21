# Model Attribution P0 Hardening

## Context

Current model attribution has two medium-risk weak points:

1. Kimi fallback depends on rigid log patterns.
2. Codex model extraction relies on broad heuristics that can misclassify metadata as model names.

This P0 focuses on extraction robustness only, without changing CLI contract.

## Scope (P0)

- Harden Kimi log fallback parsing:
  - Keep `Created new session` matching.
  - Replace single-purpose `model='...'` regex with key-value field parsing from `Using LLM model` lines.
  - Accept `model`, `model_name`, `model_id`, `modelId` keys.
  - Keep the latest model line per session.
- Harden Codex extraction strategy:
  - Remove broad recursive scan over arbitrary nested fields.
  - Use known model paths only (`model`, `model_name`, `model_id`, and selected nested paths).
  - Add candidate filtering to reject obvious placeholders (`default`, `auto`, `unknown`, rate-limit labels).

## Non-goals

- No change to alias normalization strategy in `stats/aggregator.go`.
- No change to output schema/flags.
- No change to token counting behavior.

## Acceptance Criteria

- Kimi sessions with model only in log fallback can still be attributed under flexible key-value formats.
- Kimi fallback no longer depends on exact quote style/spacing for model extraction.
- Codex parser does not attribute model from unrelated keys like `limit_name`.
- Codex parser still resolves model from existing supported payload structures.
- Existing unit/e2e behavior remains green.

## Test Cases

- `provider/kimi/parser_test.go`
  - `TestModelNameFromLogLine_Variants`
  - `TestMergeSessionModelsFromLog_UsesLatestModelForSession`
  - Existing `TestCollectSessions_ModelFallbackFromLogs`
- `provider/codex/parser_test.go`
  - `TestParseCodexSession_ModelExtractionPrefersKnownModelPath`
  - `TestParseCodexSession_ModelExtractionRejectsPlaceholder`
  - Existing extraction regression tests remain.

## Rollout

- Safe as backward-compatible parser hardening.
- If future upstream payload keys change, extend known paths list and tests in parser packages.
