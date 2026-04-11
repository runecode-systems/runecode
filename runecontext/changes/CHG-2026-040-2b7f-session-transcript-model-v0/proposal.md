## Summary
Define the minimal broker-visible session and transcript model that the first RuneCode TUI chat route should depend on instead of storing its conversation truth entirely in client-local state.

## Problem
`CHG-2026-013-d2c9-minimal-tui-v0` now includes a first-class chat route, but the control plane does not yet have a dedicated change that freezes canonical session identity, transcript structure, and session-linked references.

Without this feature, the first TUI implementation is likely to drift toward client-local-only session identity and transcript behavior.

## Proposed Change
- Define a minimal canonical session identity model.
- Define ordered transcript turn/message contracts.
- Define send-message request/response semantics or equivalent broker-mediated session interaction.
- Define links from session turns to runs, approvals, artifacts, and audit references where relevant.
- Keep the model minimal enough for the alpha TUI while strong enough to support later multi-session work.

## Why Now
This is the narrowest correct prerequisite for the TUI chat route. It should land before the first TUI implementation depends on chat/session semantics.

## Assumptions
- The first TUI feature includes a first-class chat route.
- Full multi-session management remains a later separate feature.

## Out of Scope
- Full multi-session workbench behavior.
- UI-local session switching, layout, or tab persistence.

## Impact
Creates the minimal canonical session and transcript substrate the alpha TUI needs without forcing a later rewrite away from client-local chat truth.
