## Summary
Expand RuneCode’s terminal experience beyond the MVP foundation into a pre-MVP power workspace: multi-session management, richer action-center flows, advanced live activity, deeper inspection, configurable layouts, and theme presets, all on top of the same strict brokered control-plane contracts.

## Problem
`CHG-2026-013-d2c9-minimal-tui-v0` intentionally freezes the MVP TUI foundation: dashboard-first shell, first-class chat route, strict broker-client posture, and the core interaction and visual rules.

That foundation is necessary but not sufficient for the terminal experience RuneCode ultimately needs before MVP:
- users need to manage more than one active session or workspace efficiently
- power users need faster navigation, richer inspectors, and denser workbench layouts
- operators need deeper live observability and drill-down across activity, approvals, artifacts, and audit
- users need theme presets and stronger presentation controls without destabilizing semantic state styling

If these enhancements are not planned explicitly now, they are likely to accrete as ad hoc TUI-local features that:
- treat client convenience state as control-plane truth
- overload logs as the only live observability surface
- hide important navigation or state behind clever but low-discoverability UI
- compromise the topology-neutral foundation needed for later remote or scaled backends

## Proposed Change
- Build on `CHG-2026-013-d2c9-minimal-tui-v0` with a pre-MVP advanced TUI workbench.
- Add first-class multi-session and workspace management.
- Add richer power-user navigation and command surfaces.
- Add advanced live activity, inspection, and observability views driven by typed broker/API contracts.
- Expand the action-center model to support approvals and future question-style workflows while keeping their semantics distinct.
- Add layout customization, saved workspace state, and theme presets without elevating client convenience state into trusted control-plane state.
- Add the larger post-foundation visual pass for the TUI: stronger layout hierarchy, more visual workbench composition, denser but more scannable inspectors, and presentation polish that should wait until the core route semantics and control-plane workflows stop moving.
- Preserve local-first UX while keeping the client topology-neutral for future remote or scaled backends.

## Why Now
This work belongs in `v0.1.0-beta.1`, after the alpha TUI foundation and secure model/provider-access foundations exist, but before RuneCode considers its terminal experience mature enough for MVP.

Doing this before MVP avoids a common trap: shipping a permanently minimal console and then layering advanced behavior onto it through shortcuts. This change instead keeps the same underlying trust model while making the user-facing experience strong enough for serious daily use.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode owns the end-user UX and command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Verified-mode RuneContext remains the normal operating assumption for this repository.
- `CHG-2026-013-d2c9-minimal-tui-v0` lands first and remains the base shell/foundation for this work.
- Raw model chain-of-thought remains out of scope; richer inspection focuses on typed traces, decisions, audit records, artifacts, rationale summaries, and live activity streams.
- The larger visual redesign should be sequenced after the MVP foundation and immediate broker-workflow fixes settle, so the project does not repeatedly repaint screens whose semantics are still changing.

## Out of Scope
- Replacing the MVP TUI foundation rather than extending it.
- Remote/network transport changes or alternate trust models for approvals and actions.
- Inventing pending-question or pending-answer semantics in the TUI before a canonical broker object model exists.
- Treating persisted theme, layout, or workspace UI state as trusted system state.
- Relaxing any trust-boundary, approval, policy, or audit invariant for convenience.

## Impact
This change captures the pre-MVP advanced TUI plan in one durable place, so RuneCode can grow from a strong hybrid MVP shell into a polished multi-session workbench without revisiting its foundational control-plane and trust-boundary decisions later.
