## Summary
Expand RuneCode's Bubble Tea and Lip Gloss TUI from the MVP shell into a full-screen, polished, multi-session workbench with first-class navigation, inspectors, live activity, Action Center queues, copy-friendly long-form content, saved local layouts, and theme presets, all while preserving strict broker-owned control-plane semantics and trust-boundary rules.

## Problem
`CHG-2026-013-d2c9-minimal-tui-v0` intentionally freezes the MVP TUI foundation: strict broker-client posture, a root shell plus child-route architecture, and the initial dashboard/chat/operator surfaces.

That foundation is necessary, but it is not yet the right long-term substrate for the terminal product RuneCode wants before MVP.

Current implementation gap:
- the root shell still behaves primarily as a route switcher with mostly route-local string rendering
- full-screen pane composition, overlay stacking, shared inspector/content primitives, and global watch-driven activity are not yet shell-owned capabilities
- multi-session, Action Center, and rich observability flows would otherwise accrete as per-route features instead of one coherent workbench
- transcript, diff, log, and inspector surfaces do not yet have a deliberate copy/paste model

Users now need more than a minimal console:
- a full terminal takeover with a polished software/app feel rather than a barebones shell
- clear panels, focus highlights, selected-row cues, centered command surfaces, and denser but calmer layout hierarchy
- a sidebar that is visible by default but can be shown or hidden so users can choose between visible navigation and palette-only workflows
- quick switching among canonical sessions, runs, approvals, artifacts, and audit objects
- deeper live activity and Action Center triage across approvals and operational attention
- richer inspection and drill-down flows backed by typed broker reads
- simple text copy and paste from transcripts, diffs, logs, and inspectors
- a small running animation indicator whenever canonical work is actively progressing

If these enhancements are not planned explicitly now, they are likely to accrete as ad hoc TUI-local features that:
- treat client convenience state as control-plane truth
- overload logs as the only live observability surface
- hide important navigation or state behind clever but low-discoverability UI
- compromise the topology-neutral foundation needed for later remote or scaled backends
- make copy/paste and long-form inspection worse as panes, mouse handling, and overlays multiply

## Proposed Change
- Keep Bubble Tea as the architectural backbone and Lip Gloss as the required styling/layout system.
- Extend the existing root shell plus child-model architecture rather than replacing it with a monolith.
- Make full-screen alt-screen mode the default interactive posture.
- Upgrade the root shell into a real workbench compositor with a top status bar, optional left sidebar, primary main pane, optional right inspector, bottom composer/status strip, and shell-owned overlay stack.
- Add a shared component and service layer for selectors, long-form viewports, inspectors, overlays, focus management, watch management, clipboard handling, and local workbench state.
- Make one active session in the main pane the default session model, with many sessions visible through the sidebar and quick switchers.
- Keep the sidebar visible by default, but make it easy to hide so users can rely on the command palette alone when they want more content room.
- Expand the command palette into an object-aware workbench command surface that can open routes, sessions, runs, approvals, artifacts, audit records, and shell commands.
- Standardize workbench navigation semantics around `open`, `inspect`, `jump`, and `back`.
- Add shell-owned long-lived watch streams and an ephemeral global activity cache built from typed broker watch families rather than log scraping.
- Add an `Action Center` with distinct v0 families for approvals, operational attention, and blocked work impact; reserve future question queues until a canonical broker model exists.
- Standardize a shared inspector shell with `rendered`, `raw`, and `structured` modes across sessions, runs, approvals, artifacts, and audit records.
- Treat copy/paste as first-class architecture: preserve ordinary terminal text selection, add explicit in-app copy actions, and upgrade compose input to a proper multiline paste-friendly text area.
- Persist local workbench state such as sidebar visibility, pane ratios, inspector visibility, theme preset, recents, pinned sessions, and last active session per workspace without elevating it into trusted control-plane state.
- Use an OpenCode-style fullscreen workbench level of polish as visual inspiration while preserving RuneCode's own semantics, trust posture, and information hierarchy.
- Borrow selected implementation patterns from Charmbracelet `crush` where they improve bounded rendering, reusable list and overlay primitives, and layout regression coverage, while preserving RuneCode's shell planner plus route-surface architecture.
- Defer the larger route-level visual pass until workflow semantics and first-round dogfooding stabilize, while still making the shell substrate itself feel polished from the start.
- Preserve local-first UX while keeping the client topology-neutral for future remote or scaled backends.

## Why Now
This work belongs in `v0.1.0-alpha.5`, after the alpha TUI foundation and secure model/provider-access foundations exist, but before RuneCode treats its terminal experience as MVP-ready.

Doing this before MVP avoids a common trap: shipping a permanently minimal console and then layering advanced behavior onto it through shortcuts and local-only assumptions. This change instead freezes the correct shell, navigation, activity, and copy/paste foundations now so later features can stack cleanly.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode owns the end-user UX and command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Verified-mode RuneContext remains the normal operating assumption for this repository.
- `CHG-2026-013-d2c9-minimal-tui-v0` lands first and remains the base shell/foundation for this work.
- Raw model chain-of-thought remains out of scope; richer inspection focuses on typed traces, decisions, audit records, artifacts, rationale summaries, and live activity streams.
- Planning reviewed OpenCode-style fullscreen workbench references and the Bubble Tea/Lip Gloss ecosystem as positive directional input for the shell and visual foundation.
- Planning and follow-up implementation review also examined Charmbracelet `crush` directly. The review concluded that `crush` shares the same Go, Bubble Tea, and Lip Gloss ecosystem, but uses a more component-local, rectangle-bounded draw model than RuneCode's shell compositor. RuneCode should adopt the bounded-component discipline and regression-test posture from `crush` without rewriting the workbench around a different root rendering architecture.
- The larger visual redesign should be sequenced after the shell substrate, route semantics, and immediate broker-workflow fixes settle, so the project does not repeatedly repaint screens whose meaning is still changing.

## Out of Scope
- Replacing the MVP TUI foundation rather than extending it.
- Replacing Bubble Tea or Lip Gloss with a different UI foundation.
- Rewriting the workbench around `crush`'s screen-buffer and rectangle-draw architecture.
- Replacing the shell planner plus route-surface contracts with a chat-first monolithic UI model.
- Remote/network transport changes or alternate trust models for approvals and actions.
- Inventing pending-question or pending-answer semantics in the TUI before a canonical broker object model exists.
- Treating persisted theme, layout, recents, pinned sessions, or workspace UI state as trusted system state.
- Sacrificing ordinary terminal text selection in the name of richer pane or mouse behavior.
- Requiring equal multi-session pane layouts in the first advanced workbench cut.
- Relaxing any trust-boundary, approval, policy, or audit invariant for convenience.

## Impact
This change captures the pre-MVP advanced TUI plan in one durable place, freezes the shell and interaction clarifications that later work should inherit, and gives RuneCode a path from a strong hybrid MVP shell to a polished multi-session workbench without revisiting its foundational control-plane and trust-boundary decisions later.
