## Summary
Deliver the first real RuneCode terminal experience as a hybrid local TUI: a dashboard-first ops console with a first-class chat/coding route, all backed by the broker local API, typed approval flows, typed audit views, and the real secure control-plane surfaces.

## Problem
RuneCode needs an end-user terminal interface that feels modern, colorful, dense, and fast without introducing a shortcut UX layer that bypasses the project’s trust boundaries, policy semantics, approval model, or audit model.

If the first TUI is defined too narrowly as only an operator console:
- the later chat/coding experience will end up bolted on as a separate mental model
- session and live-activity foundations will drift into client-local state rather than broker-visible contracts
- future remote or scaled control-plane access will be harder to support without revisiting the TUI substrate

If the first TUI is defined too loosely:
- the client may start inventing authorization, workflow, or status semantics locally
- users may get a smooth-looking interface that quietly violates RuneCode’s strict core tenants
- later CLI, remote-client, and observability work will have to unwind UI-specific shortcuts

## Proposed Change
- Make `CHG-2026-013-d2c9-minimal-tui-v0` the MVP TUI foundation, not just a thin review console.
- Start the TUI in a dashboard/ops-console route while making chat/coding a first-class route in the same shell.
- Use Bubble Tea and its message-driven architecture as the required framework and implementation posture.
- Keep the TUI a least-privilege broker client that consumes typed read/write/stream contracts rather than daemon-private data or scraped CLI output.
- Freeze the UX and data-contract expectations needed for a strong foundation:
  - hybrid dashboard + chat shell
  - keyboard-first interaction with mouse support
  - visible primary navigation on wide layouts
  - typed approval, run, artifact, audit, status, and live-activity surfaces
  - semantic theming and colorful but non-decorative presentation
  - explicit authoritative vs advisory state handling
- Capture known contract follow-ups that must be handled through broker/API work rather than through TUI-local heuristics:
  - richer approval detail/read-model support
  - audit record drill-down support
  - typed watch/event surfaces for live activity
  - minimal canonical session/transcript model to support the chat route
- Keep advanced multi-session, power-user workspace management, richer inspection, and enhanced observability in a dedicated pre-MVP follow-on change.

## Why Now
This work remains scheduled for `v0.1.0-alpha.3` because RuneCode needs the first honest user-facing secure slice through the real brokered local API, policy, audit, artifact, and approval surfaces.

The goal is not to ship the fastest possible TUI. The goal is to define the best durable foundation so later TUI features, CLI work, and future remote or scaled client topologies can build on one typed control-plane substrate without revisiting fundamental UX or trust-boundary decisions.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode owns the end-user UX and command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Verified-mode RuneContext remains the normal operating assumption for this repository.
- MVP remains local-first, but boundary-visible contracts must remain topology-neutral so later remote or scaled backends can reuse the same semantics.
- Raw model chain-of-thought is not an MVP artifact concept; inspectability focuses on typed traces, decisions, approvals, artifacts, and rationale summaries where defined.

## Out of Scope
- Full multi-session and power-user workspace management.
- User-customizable themes and saved layout presets.
- Pending-question or pending-answer flows before a canonical broker object model exists for them.
- Remote/network client transport changes.
- Any relaxation of approval, policy, audit, or trust-boundary semantics for UX convenience.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
This change establishes the durable TUI foundation for RuneCode: one visible shell, one secure control-plane contract model, one set of interaction principles, and one path from local MVP to richer pre-MVP enhancements without rewriting the base architecture later.
