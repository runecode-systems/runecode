# Tasks

## Multi-Session And Workspace Management

- [ ] Add first-class multi-session browsing and switching on top of the MVP session foundation.
- [ ] Add session summaries that expose enough metadata for quick switching without opening each session.
- [ ] Support workspace-level grouping or switching where useful.
- [ ] Support background session awareness so users can tell which sessions are active, blocked, or waiting.
- [ ] Preserve stable links between sessions and related runs, approvals, artifacts, and audit views.
- [ ] Keep session identity canonical and broker-aligned rather than client-local-tab-aligned.

Parallelization: depends on the MVP session foundation and any related broker/session read-model work.

## Layout, Inspectors, And Saved Workspace State

- [ ] Add multi-pane inspector and split-view workspace layouts for wide terminals.
- [ ] Add resizable panes where practical.
- [ ] Add inspector stacks or equivalent drill-down patterns for keeping context while exploring detail.
- [ ] Add saved layout presets and restoration of prior workspace arrangement.
- [ ] Keep all persisted layout/workspace state explicitly non-authoritative.
- [ ] Ensure layouts degrade cleanly on narrower terminals.

Parallelization: layout and persistence work can proceed in parallel with session and inspection work once shell extension points are stable.

## Action Center Expansion

- [ ] Expand the queue/worklist model into an `Action Center` route.
- [ ] Keep approvals as a distinct queue family with fast filtering and triage.
- [ ] Integrate future question/pending-answer queue families only when a canonical broker object model exists.
- [ ] Keep approvals and future questions visibly distinct inside the same action center.
- [ ] Surface urgency, expiry, staleness, supersession, and blocked-work impact clearly.
- [ ] Support fast keyboard triage and queue navigation.

Parallelization: approvals can evolve in parallel; question integration depends on separate control-plane model work.

## Advanced Live Activity And Observability

- [ ] Add a richer live activity workspace that can fan in updates across sessions, runs, approvals, and other queue families.
- [ ] Consume typed watch/event families rather than relying on polling plus logs only.
- [ ] Surface what is active, blocked, waiting, degraded, or completed without forcing users to open every route manually.
- [ ] Support drill-down from live summary items into their full inspectors.
- [ ] Keep logs supplementary rather than the primary live operator surface.

Parallelization: depends on watch/event contract work in the broker/API lane.

## Rich Inspection Surfaces

- [ ] Add deeper transcript and activity inspectors for sessions.
- [ ] Add richer approval inspection and history views.
- [ ] Add policy-decision drill-down views where broker-visible data supports it.
- [ ] Add gate/evidence and override inspection improvements.
- [ ] Add audit record drill-down and linked-reference navigation using broker-owned reads.
- [ ] Add rendered/raw/structured presentation modes across long-form inspection surfaces where useful.
- [ ] Add canonical cross-links so users can pivot quickly among sessions, runs, approvals, artifacts, and audit records.

Parallelization: different inspectors can be implemented in parallel once the required typed detail reads exist.

## Power-User Navigation And Commands

- [ ] Expand the command palette into a broader command surface for the workbench.
- [ ] Add fast session/run/approval/artifact quick switching.
- [ ] Add high-frequency shortcuts for common navigation and triage flows.
- [ ] Support slash-command style interaction where it fits naturally in the chat/workbench model.
- [ ] Keep help output generated from the real keymap definitions.
- [ ] Preserve discoverability while improving speed for expert users.

Parallelization: command and shortcut work can proceed in parallel with route and inspector work.

## Theme Presets And Presentation Controls

- [ ] Add theme presets built on semantic tokens.
- [ ] Preserve semantic meaning across all presets.
- [ ] Support stronger presentation controls for markdown, code, diffs, logs, and structured objects.
- [ ] Preserve raw and structured inspection modes alongside rendered modes.
- [ ] Keep color as expressive and semantic but never the only cue.

Parallelization: can proceed in parallel once the semantic token foundation from the MVP change is stable.

## Topology-Neutral Client Posture

- [ ] Ensure advanced workbench behavior does not depend on host-local storage identity, socket names, or daemon-private details.
- [ ] Keep persisted convenience state local and non-authoritative.
- [ ] Preserve the ability for the same workbench semantics to target local or future remote/scaled backends through the same logical contracts.

Parallelization: architectural guardrails should be maintained continuously across implementation work.

## Acceptance Criteria

- [ ] Users can manage and switch among multiple sessions/workspaces efficiently in the TUI.
- [ ] Users can observe active, blocked, waiting, and degraded work through richer live activity surfaces without depending on log scraping.
- [ ] Users can drill into approvals, artifacts, audit records, and linked control-plane objects through typed broker-owned inspection flows.
- [ ] Power users gain materially faster navigation and command workflows without making the product undiscoverable to ordinary users.
- [ ] Theme presets and saved layouts remain convenience features and do not alter semantic control-plane meaning.
