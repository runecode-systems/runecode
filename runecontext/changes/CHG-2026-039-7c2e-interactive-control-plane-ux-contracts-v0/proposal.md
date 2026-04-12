## Summary
Track the broker/API-side control-plane UX contracts that must be implemented before the first RuneCode TUI feature so the alpha terminal client can rely on canonical session, approval, audit, and live-activity surfaces instead of inventing those semantics locally.

## Problem
`CHG-2026-013-d2c9-minimal-tui-v0` now depends on several control-plane surfaces that are too important to leave implicit:
- session and transcript identity
- approval review detail models
- audit record drill-down
- typed live watch streams

If these are not captured as explicit feature changes and implemented first, the alpha TUI will be pushed toward heuristics, local-only state, and log-driven approximations that would have to be undone later.

## Proposed Change
- Keep this change as the project-level tracker for interactive control-plane UX contracts.
- Track `CHG-2026-040-2b7f-session-transcript-model-v0` for canonical session/transcript contracts.
- Track `CHG-2026-041-4d8a-approval-review-detail-models-v0` for richer approval review detail models.
- Track `CHG-2026-042-6f3c-audit-record-drill-down-v0` for typed audit record drill-down.
- Track `CHG-2026-043-8e9b-live-activity-watch-streams-v0` for typed run/approval/session watch streams.
- Freeze the rule that these contracts must land before the first TUI implementation depends on them.

## Why Now
This work is the narrowest correct way to set the TUI foundation up properly. It creates one project-level place to track the broker/API contract work that should precede `CHG-2026-013-d2c9-minimal-tui-v0` implementation.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode owns the end-user UX and command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Verified-mode RuneContext remains the normal operating assumption for this repository.

## Out of Scope
- Implementing the TUI itself.
- Treating UI-local behavior as an acceptable substitute for missing control-plane contracts.
- Remote/network transport changes.

## Impact
Creates one project-level anchor for the contract work that the alpha TUI should depend on, making the required sequencing explicit and reviewable.
