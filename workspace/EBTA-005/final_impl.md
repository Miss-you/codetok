# Final Implementation

Implement native Kimi usage events as described in `final_impl_v1.md`.

No command behavior, README content, or cross-provider collector behavior changes in this task. Those are later EBTA tasks.

Review follow-up:

- Kimi events now populate `EventID` from `message_id` when available, otherwise from source line number.
- Tests also cover metadata model priority over wire payload model, and wire payload model priority over log fallback.

OpenSpec change: `event-based-token-aggregation-kimi-events`.
