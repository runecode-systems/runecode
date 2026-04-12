## Summary
Define typed live activity watch streams for runs, approvals, and sessions so the first RuneCode TUI can build real live UX on canonical event surfaces instead of relying primarily on polling and logs.

## Problem
The alpha TUI needs a live foundation for active work, approvals, and chat/session activity. Without a dedicated feature, the client is likely to fall back to polling plus logs as its primary live UX substrate.

## Proposed Change
- Define typed watch/event families for runs, approvals, and sessions.
- Preserve explicit stream identity, ordering, and terminal-state rules.
- Define the minimum live event surface the alpha TUI needs.
- Keep logs supplementary rather than the primary live operator truth.

## Why Now
This is a prerequisite for the alpha TUI to have a strong live foundation instead of a temporary approximation that would need redesign later.

## Assumptions
- Existing stream semantics from the broker local API remain the base model.
- New watch families should be additive and consistent with existing stream rules.

## Out of Scope
- One generic event bus.
- Replacing typed logs and artifact reads with one untyped stream mechanism.

## Impact
Creates the live activity substrate the alpha TUI should depend on for runs, approvals, and sessions without drifting into log-centric operator UX.
