# Cursor Token Research Design

## Goal

Determine the most credible way for `codetok` to support Cursor token accounting, with a focus on Cursor agent/composer usage first and Tab usage only if there is a defensible data source.

## Current `codetok` constraints

- `codetok` currently aggregates usage from local session artifacts only.
- `codetok` does not call provider APIs today.
- Existing providers fit a simple pattern: discover files on disk, parse token counters, aggregate by day/session.

That matters because Cursor does not expose an obvious local JSONL session log comparable to Claude Code or Codex CLI.

## Research summary

### 1. Best open-source reference for Cursor token totals: `tokscale`

Repository:
- https://github.com/junhoyeo/tokscale

What it does:
- Authenticates with a `WorkosCursorSessionToken`.
- Calls Cursor dashboard endpoints directly.
- Caches the returned CSV under `~/.config/tokscale/cursor-cache/`.
- Parses the cached CSV as local data after sync.

Relevant implementation:
- `crates/tokscale-cli/src/cursor.rs`
  - `USAGE_CSV_ENDPOINT = https://cursor.com/api/dashboard/export-usage-events-csv?strategy=tokens`
  - `USAGE_SUMMARY_ENDPOINT = https://cursor.com/api/usage-summary`
- `crates/tokscale-core/src/sessions/cursor.rs`
  - Parses Cursor-exported CSV rows into token records.

Important CSV shape observed in Tokscale tests:

```csv
Date,Kind,Model,Max Mode,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens,Total Tokens,Cost
"2025-11-13T18:36:05.846Z","Included","auto","No","28342","775","105891","21282","156290","0.19"
"2025-11-13T13:35:04.658Z","On-Demand","gpt-5-codex","No","0","8263","66964","1612","76839","0.03"
```

What this tells us:
- Cursor does have an authoritative token export surface.
- The export is already rich enough for `codetok`'s token model: input, cache read, output, total, cost.
- The visible split is billing-oriented (`Kind`, `Max Mode`), not obviously `agent` vs `tab`.

Implication:
- Tokscale is the strongest reference if the goal is accurate Cursor token totals.
- It is not a pure local-log parser. It depends on an explicit sync step against Cursor's web API.

### 2. Best open-source reference for future-only interception: `Cursor Lens`

Repository:
- https://github.com/HamedMP/CursorLens

What it does:
- Sits in front of Cursor as a proxy by overriding Cursor's OpenAI base URL.
- Logs request and response metadata to its own database.
- Computes token and cost analytics from proxied traffic.

Relevant implementation:
- `src/app/[...openai]/route.ts`
  - reads provider `usage`
  - persists `inputTokens`, `outputTokens`, `totalTokens`, and cost metadata

Implication:
- Cursor Lens is a useful reference for live instrumentation.
- It is not a good fit for `codetok`'s current model because it only sees future traffic and requires a proxy deployment.
- It also does not solve post-hoc local-history import.

### 3. Cursor local artifacts inspected on this machine

#### A. `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb`

Observed keys:
- `composerData:*`
- `bubbleId:*`
- `agentKv:*`

Findings:
- `composerData:*` stores conversation state and context selection data.
- `bubbleId:*` records include a `tokenCount` object, but sampled rows on this machine were `0/0`.
- I did not find reliable non-zero token totals here.

Conclusion:
- This database may help recover chat/session metadata.
- It is not a credible primary source for Cursor billing tokens.

#### B. `~/Library/Application Support/Cursor/logs/.../Cursor Tab.log`

Findings:
- Contains request IDs, TTFT/stream timing, and debug output.
- I did not find token counts in sampled log lines.

Conclusion:
- Useful for performance/debug telemetry.
- Not suitable for Tab token accounting.

#### C. `~/.cursor/ai-tracking/ai-code-tracking.db`

Observed tables:
- `ai_code_hashes`
- `scored_commits`
- `conversation_summaries`
- `tracked_file_content`
- `ai_deleted_files`
- `tracking_state`

Important schema evidence:
- `scored_commits` contains `tabLinesAdded`, `tabLinesDeleted`, `composerLinesAdded`, `composerLinesDeleted`
- `ai_code_hashes.source` distinguishes sources such as `composer`

Findings:
- This database is about accepted-code attribution, not token accounting.
- It is useful for `tab` vs `composer/agent` line attribution.
- It does not contain token columns.

Conclusion:
- This is a strong secondary signal if we ever want Cursor activity attribution.
- It does not solve token totals.

### 4. Official Cursor surfaces

Relevant docs:
- Pricing / usage docs say users can view usage and token breakdowns in the Cursor dashboard.
- The AI code tracking API returns accepted lines split by categories such as `TAB` and `COMPOSER`, plus model metadata.

References:
- https://cursor.com/docs/pricing/usage-based-pricing
- https://cursor.com/docs/admin/teams/ai-code-tracking/api-reference

Implication:
- Cursor's own official split for `tab` vs `composer` appears to be code attribution, not token export.
- That aligns with the local `ai-code-tracking.db` evidence.

## Options for `codetok`

### Option A: Recommended

Add Cursor support through a manual sync/import workflow:

1. Introduce a `cursor` provider that reads cached/imported Cursor usage CSV files from disk.
2. Keep day/session aggregation local after the file exists.
3. Add a separate explicit sync command later if desired, for example:
   - `codetok cursor sync`
   - authenticate with a user-supplied Cursor session token
   - fetch the dashboard CSV
   - save it under a `codetok`-owned cache directory

Why this is the best fit:
- Reuses the existing parser/aggregation architecture.
- Keeps normal reporting local-file based.
- Matches the best available reference implementation.
- Avoids pretending that Cursor local logs already contain what they do not.

Trade-offs:
- Breaks the repo's current strict no-provider-API stance unless sync is framed as an explicit opt-in import step.
- Requires auth UX and secure token handling.
- Still may not separate `agent` from `tab` unless Cursor's export exposes that distinction.

### Option B: Secondary enhancement

Use `~/.cursor/ai-tracking/ai-code-tracking.db` only as a supplemental attribution source.

Possible use:
- annotate Cursor reports with accepted-line metrics such as:
  - composer lines
  - tab lines

Why it is useful:
- Gives a credible way to report Tab-related activity.
- Aligns with Cursor's own AI code tracking concepts.

Why it is not enough:
- No token totals.
- No evidence that it can reconstruct billing usage.

### Option C: Not recommended

Attempt pure local parsing from `state.vscdb`, `Cursor Tab.log`, and other Cursor app logs for token totals.

Why not:
- Local evidence was weak.
- The strongest sampled fields were zero-valued or timing-only.
- This path would likely be brittle and undercount or miscount.

## Recommendation

The next implementation should target Cursor agent/composer token totals through imported dashboard CSV data, not through raw local Cursor app logs.

Concretely:
- Phase 1: support parsing Cursor-exported CSV from disk.
- Phase 2: optionally add `codetok cursor sync` to fetch and cache that CSV with explicit user consent.
- Phase 3: if the product goal still cares about Tab, add a separate non-token activity view backed by `ai-code-tracking.db` line attribution.

## Expected product outcome

If we follow the recommended path:
- `codetok` can support credible Cursor token totals.
- The implementation can stay close to the existing provider pattern.
- `tab` token counting remains unsupported unless Cursor exposes a stronger source than the ones found here.
- The best near-term `tab` story is activity attribution, not token accounting.

## References

- Tokscale repository: https://github.com/junhoyeo/tokscale
- Cursor Lens repository: https://github.com/HamedMP/CursorLens
- Cursor usage-based pricing docs: https://cursor.com/docs/pricing/usage-based-pricing
- Cursor AI code tracking API docs: https://cursor.com/docs/admin/teams/ai-code-tracking/api-reference
