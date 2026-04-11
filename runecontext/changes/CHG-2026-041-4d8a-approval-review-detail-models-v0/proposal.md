## Summary
Define richer broker-projected approval review detail models so the first RuneCode TUI can explain approvals correctly without scraping payloads or flattening important approval distinctions.

## Problem
The alpha TUI needs more than `ApprovalSummary` plus optional raw envelopes to deliver a correct approval review experience. Without a dedicated feature, the client will be pushed toward local heuristics and payload inspection.

## Proposed Change
- Define richer approval review detail models.
- Surface `policy_reason_code` directly in broker-projected review models.
- Surface exact-action vs stage-sign-off binding kind explicitly.
- Surface structured “what changes if approved” and blocked-work scope.
- Surface stale, superseded, expired, consumed, approved, and denied review semantics through typed fields and reason codes.

## Why Now
This is a prerequisite for the alpha TUI approval review experience to be both usable and faithful to the project’s approval model.

## Assumptions
- Approval creation remains policy-derived and broker-materialized.
- Approval payloads remain signed and authoritative, but the TUI should not need to inspect low-level payload structure to explain approvals safely.

## Out of Scope
- Approval profile expansion beyond MVP `moderate`.
- Inventing a second approval truth separate from policy and broker materialization.

## Impact
Creates the approval-detail surface the alpha TUI should depend on so approval review can stay typed, structured, and aligned with exact-action and stage-sign-off semantics.
