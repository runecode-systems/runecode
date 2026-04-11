# Design

## Overview
Extend the MVP RuneCode TUI into a pre-MVP multi-session, power-user workspace while preserving the same Bubble Tea shell architecture, the same broker-owned control-plane semantics, and the same strict trust-boundary model defined for the MVP foundation.

This change is not a replacement for `CHG-2026-013-d2c9-minimal-tui-v0`. It is the next layer of user-facing capability on top of that foundation.

## Product Shape

### Workbench Identity
- The TUI evolves from a hybrid dashboard-plus-chat shell into a true terminal workbench.
- Dashboard remains an entry route, but no longer carries the burden of representing the whole product alone.
- Users should be able to move fluidly among active sessions, runs, action queues, artifacts, and audit inspectors without feeling that they are leaving one disconnected screen family for another.

### Power-User Goal
- Keep the default experience discoverable.
- Add power-user speed through shortcuts, quick switching, command surfaces, and dense layouts.
- Avoid turning the TUI into a hidden-knowledge product that only works well for experts.

## Key Decisions
- This change extends the MVP TUI foundation rather than superseding it.
- Bubble Tea remains the required framework and architectural posture.
- The root shell model plus child route/component model architecture should continue; advanced behavior should not collapse back into one monolithic workbench model.
- Multi-session and workspace management must be built on canonical session identities and broker-visible/session-contract semantics rather than client-local tab identity.
- Persisted TUI workspace state such as open panes, selected inspectors, theme preset, and layout preference is convenience state only. It must not be used as control-plane authority.
- Advanced observability should come from typed watch/event families and typed detail reads, not from log scraping and not from daemon-private storage access.
- One `Action Center` route may present multiple queue families, but approvals and future questions must remain semantically distinct sections backed by distinct canonical models.
- If a pending-question or pending-answer control-plane model does not yet exist, this TUI change must not invent one locally.
- Rich inspection should focus on structured activity/decision traces, artifacts, approvals, policy decisions, gate evidence, session turns, audit records, and rationale summaries where defined.
- Raw chain-of-thought remains out of scope.
- Theme presets and layout customization are allowed, but semantic theme tokens remain the foundation and color remains non-exclusive as a cue.
- The advanced workbench must stay topology-neutral so the same logical client behavior can later target a local backend or a vertically/horizontally scaled remote backend through the same control-plane semantics.

## Multi-Session Model

### Sessions As First-Class UX Objects
- The workbench should support more than one active or recently active session.
- Session switching should be quick, visible, and keyboard-friendly.
- Session summaries should expose enough typed metadata to help users choose the right session without opening each one.
- Sessions should remain linked to their relevant runs, artifacts, approvals, and audit references.

### Workspace Management
- The TUI should support workspace-level grouping or switching where useful.
- Session and workspace views should not require users to memorize hidden navigation patterns.
- Quick switchers and command surfaces should make it possible to jump directly to:
  - a session
  - a run
  - an approval
  - an artifact
  - an audit record or view

## Layout And Workbench Model

### Layout Principles
- Wide terminals should be able to show multiple related panes without overwhelming the user.
- Narrow terminals should degrade cleanly into simpler routed views.
- Primary navigation and shell state must remain discoverable even when layouts become denser.

### Advanced Layout Capabilities
- Support inspector panes and split-view workbench layouts.
- Support resizable pane layouts where practical.
- Support saved layout presets and restoration of prior workspace arrangement.
- Keep layout persistence local and non-authoritative.

## Action Center Model

### Queue Structure
- The advanced workbench should group pending interactive work in one `Action Center` route.
- `Approvals` remain their own canonical queue family.
- Future `Questions` or pending-answer flows should appear as a separate queue family inside the same route once a canonical broker model exists.

### Queue Behavior
- Surface counts, urgency, expiry, stale/superseded posture, and blocked-work impact.
- Support fast keyboard triage and quick switching between queue items.
- Preserve semantic distinctions between:
  - approval-required actions
  - future answer-required prompts
  - system errors
  - blocked-but-non-interactive waits

## Advanced Live Activity And Observability

### Live Activity Principles
- The workbench should support a richer live activity view than the MVP routes alone.
- Live activity should fan in typed events across sessions, runs, approvals, and other queue families.
- Users should be able to see what is active, blocked, waiting, or degraded without manually opening each route and polling for changes.

### Event And Stream Expectations
- Advanced observability depends on typed watch/event families such as:
  - `RunWatchEvent`
  - `ApprovalWatchEvent`
  - `SessionWatchEvent`
  - future question/watch families where a canonical model exists
- The workbench should preserve stream identity, ordering guarantees, and terminal-state clarity rather than inferring meaning from log text.

## Rich Inspection Model

### Inspection Goals
- Users should be able to drill from summary to detail quickly and safely.
- Inspectors should support rendered, raw, and structured views where appropriate.
- Cross-linking should be based on canonical identifiers and typed references, not guessed relationships.

### Inspection Surfaces
The advanced workbench should deepen inspection for:
- session transcripts and linked activity
- approvals and their resolution history
- policy decisions and explanation surfaces
- run/stage/role/gate progress
- artifacts, diffs, logs, and evidence
- audit records and audit verification posture
- rationale summaries and decision/activity traces where the underlying control-plane model exists

### Audit Drill-Down
- Audit inspection should continue to use broker-owned derived views and typed detail reads.
- The TUI must not read ledger files directly or treat local storage as a UI-facing API.

## Power-User Interaction Model

### Command Surface
- Expand the command palette into a broader workbench command surface.
- Support quick switching and direct open-by-identity flows where practical.
- Support slash-command style interaction where it fits naturally in the chat/workbench model.

### Shortcut Coverage
- Expand the shared shortcut registry so high-frequency work does not require repeated route switching through visible nav alone.
- Preserve discoverability through generated help and visible hinting.
- Avoid route-specific shortcut chaos.

## Visual And Theme Model

### Presentation Goals
- Retain the colorful, professional, dense visual language from the MVP foundation.
- Add theme presets and user-selectable presentation modes.
- Preserve semantic consistency across presets.

### Theme Rules
- Themes should be built on semantic tokens, not direct view-level color choices.
- A theme preset may change expression, but not semantic meaning.
- Content viewers should support strong syntax and markdown presentation without sacrificing raw and structured inspection modes.

## Future Topology Neutrality
- The workbench must continue to avoid local-only semantic assumptions.
- Session identity, approval identity, audit record identity, and other user-facing control-plane objects must remain topology-neutral.
- This change does not add remote transport, but it must preserve the path for a local TUI to later target a vertically or horizontally scaled backend through the same logical contracts.

## Foundation Shortcuts To Avoid
- Do not implement multi-session as client-local tabs detached from canonical session identity.
- Do not make advanced observability a log viewer dressed up as a control-plane inspector.
- Do not invent pending-question semantics locally before a typed canonical model exists.
- Do not let saved layouts, theme state, or last-opened inspectors become accidental authority inputs.
- Do not hide primary workbench navigation behind a hamburger-first shell.
- Do not replace structured inspection with prose-heavy summaries when canonical typed data exists.

## Main Workstreams
- Multi-Session And Workspace Management
- Action Center Expansion
- Advanced Live Activity And Observability
- Rich Inspection Surfaces
- Power-User Navigation And Commands
- Theme Presets And Layout Customization

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches sessions, approvals, audit, observability, topology, or typed contracts, the plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
