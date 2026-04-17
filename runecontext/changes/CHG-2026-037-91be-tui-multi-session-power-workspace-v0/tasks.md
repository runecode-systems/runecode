# Tasks

## Phase 1: Fullscreen Workbench Shell

- [x] Enable alt-screen as the default interactive launch posture.
- [x] Replace the current header-plus-route rendering shell with a real shell compositor that owns the top status bar, optional left sidebar, primary main pane, optional right inspector, bottom composer or status strip, and overlay stack.
- [x] Make the sidebar visible by default and add a one-key toggle so users can hide it when they want palette-only navigation or more content room.
- [x] Standardize wide, medium, and narrow breakpoint behavior in the shell rather than in individual routes.
- [x] Move child routes toward shell-owned main-pane and inspector contracts instead of full-screen string rendering.
- [x] Add shell-owned breadcrumbs, backstack, and status surfaces.

Parallelization: this phase should land before broad route migration or deep visual work.

## Phase 2: Shared Component And Service Layer

- [x] Add reusable selectors or directories, long-form viewports, inspector headers, tabs or mode switches, empty/loading/error states, and centered overlay surfaces.
- [x] Borrow `crush`-style component-local bounded rendering patterns for shared lists and overlays without replacing RuneCode's shell planner plus route-surface architecture.
- [x] Add a shell-owned focus manager.
- [x] Add a shell-owned overlay manager.
- [x] Add a shell-owned command registry.
- [x] Add a shell-owned clipboard service.
- [x] Add a shell-owned workbench state store for local convenience state.
- [x] Add a shell-owned toast or status service.
- [x] Use Bubble Tea child models plus selective Bubbles primitives (`viewport`, `textarea`, `spinner`, `help`) where they accelerate the foundation without forcing generic list or table UX.

Parallelization: may begin as Phase 1 stabilizes, but shared interfaces should freeze before multiple routes migrate.

## Phase 3: Session Workspace Foundation

- [x] Add a session directory or sidebar and quick switcher built on canonical session identities.
- [x] Default the main workbench to one active session in the main pane.
- [x] Show the minimum background session awareness metadata: canonical session ID or label, workspace ID, last activity time, last activity kind, preview text, incomplete-turn state, high-level state cue, linked run count, and linked approval count.
- [x] Add local convenience markers such as recents, pinned sessions, and new activity since viewed without elevating them into authority.
- [x] Preserve canonical links from sessions to runs, approvals, artifacts, and audit records.
- [x] Keep equal multi-session pane layouts out of the initial foundation while ensuring the shell does not block them on future wide-terminal work.

Parallelization: depends on Phase 1 and typed session contracts.

## Phase 4: Object Navigation And Command Surface

- [x] Expand the command palette into an object-aware workbench command surface.
- [x] Support quick opening or switching of routes, sessions, runs, approvals, artifacts, audit records, and shell commands such as pane toggles and layout actions.
- [x] Standardize `open`, `inspect`, `jump`, and `back` behavior across the workbench.
- [x] Ensure palette-only navigation remains fully capable when the sidebar is hidden.
- [x] Keep help and shortcut documentation generated from the real keymap definitions.
- [x] Preserve discoverability while increasing speed for expert users.

Parallelization: may overlap late Phase 3, but standard navigation verbs should freeze before widespread cross-linking work.

## Phase 5: Shell-Owned Watch Manager And Live Activity Cache

- [x] Add shell-owned long-lived follow watchers for `RunWatchEvent`, `ApprovalWatchEvent`, and `SessionWatchEvent`.
- [x] Add an ephemeral shell-owned presentation cache of broker-projected summaries derived from those watch streams.
- [x] Derive a global live activity feed from the watch-backed cache.
- [x] Surface explicit shell sync health so users can tell when live activity is degraded or disconnected.
- [x] Keep watch ownership in the shell rather than duplicating watch logic per route.

Parallelization: depends on typed watch/event contract work.

## Phase 6: Running Indicator And Activity Semantics

- [x] Add shell-level activity semantics for `loading`, `running`, and `degraded sync`.
- [x] Show a small animated running indicator in the status bar whenever canonical work is actively progressing.
- [x] Add row-level or pane-level activity markers for the focused or actively running object.
- [x] Keep running semantics driven by typed broker-visible activity rather than local timers alone.

Parallelization: can begin once Phase 5 exists.

## Phase 7: Action Center

- [x] Add an `Action Center` route or shell surface that groups interactive and operator-attention queues without collapsing their semantics together.
- [x] Keep approvals as a distinct canonical queue family.
- [x] Add operational attention surfaces for degraded audit posture, anchoring problems, watch disconnects, and comparable operator-facing issues.
- [x] Add blocked-work impact cues as a separate family.
- [x] Reserve question or answer queues for future canonical broker work; do not invent them locally.
- [x] Support fast keyboard triage, counts, urgency, expiry, stale or superseded cues, and blocked-work impact drill-down.

Parallelization: approvals can evolve in parallel; question integration waits on separate control-plane model work.

## Phase 8: Shared Inspector And Long-Form Content Foundation

- [x] Add a standard inspector shell with summary header, identity or status badges, linked references, local actions, and a `rendered`/`raw`/`structured` mode switch.
- [x] Migrate session transcript inspection onto the shared inspector shell.
- [x] Migrate run, approval, artifact, and audit inspection onto the shared inspector shell.
- [x] Add stable long-form viewport handling for transcripts, diffs, logs, markdown, and raw structured content.
- [x] Preserve canonical cross-links among sessions, runs, approvals, artifacts, and audit records.
- [x] Keep audit drill-down on broker-owned typed reads rather than private storage access.

Parallelization: individual object inspectors can migrate in parallel after the shared shell exists.

## Phase 9: Copy, Paste, And Selection UX

- [x] Add a visible selection mode that reduces or disables mouse capture so drag-to-select works normally.
- [x] Ensure terminal text selection remains a supported first-class copy path for long-form content.
- [x] Add explicit in-app copy actions for canonical IDs, digests, raw blocks, transcript excerpts, artifact previews, and linked references.
- [x] Add clipboard or OSC52 integration when available, without making it the only copy path.
- [x] Replace ad hoc compose input handling with a proper text area model that supports multiline paste and bracketed paste.
- [x] Ensure no core interaction requires mouse drag in a way that sacrifices normal text selection.

Parallelization: may begin once shared content surfaces and the bottom strip model exist.

## Phase 10: Local Persistence, Layout, And Themes

- [x] Persist sidebar visibility, pane ratios or collapsed states, inspector visibility, preferred presentation mode, theme preset, last active session per workspace, recent objects, and pinned sessions as local-only convenience state.
- [x] Optionally persist the last-opened primary route if it does not blur canonical state boundaries.
- [x] Key persisted state by logical broker target plus canonical workspace or session identifiers when relevant.
- [x] Keep host-local details such as socket paths out of semantic identity.
- [x] Add semantic theme tokens for surfaces, borders, focus, selection, text tiers, overlays, and semantic states.
- [x] Add theme presets that vary expression without changing meaning.
- [x] Preserve non-color cues across themes.
- [x] Support resizable wide-terminal panes and clean restoration of saved layout arrangements.

Parallelization: can proceed in parallel with inspector migration after shell contracts are stable.

## Phase 11: Responsive Degradation

- [x] Define and implement standardized wide, medium, and narrow terminal behaviors in the shell.
- [x] On wide terminals, support sidebar plus main pane plus inspector where useful.
- [x] On medium terminals, collapse one secondary pane while preserving primary navigation and quick inspect or open actions.
- [x] On narrow terminals, degrade sidebar and inspector into overlays or full-screen views rather than route-local hacks.
- [x] Verify the workbench remains fully navigable when the sidebar is hidden or automatically collapsed.

Parallelization: should track shell and inspector work continuously.

## Phase 12: Deferred Larger Visual Pass

- [x] After route semantics, Action Center semantics, dashboard data expectations, and first-round dogfooding stabilize, deepen the visual workbench pass.
- [x] Strengthen pane framing, hierarchy, spacing, focus affordances, and summary-to-detail transitions so the TUI feels like a polished terminal application rather than formatted route text.
- [x] Adopt `crush`-informed refinements for bounded overlay composition, reusable list behavior, and resize or clipping regression tests while keeping RuneCode's existing shell architecture.
- [x] Refine diff, markdown, and structured viewers plus inspector grouping for lower cognitive load.
- [x] Capture repeatable screenshots or VHS tapes for the key workbench flows.
- [x] Keep the larger visual pass explicitly downstream of semantic stabilization so the project does not repeatedly repaint moving workflows.

Parallelization: should begin only after the earlier foundation phases stabilize.

## Acceptance Criteria

- [x] The TUI launches as a full-screen workbench in alt-screen mode and no longer feels like a barebones shell.
- [x] Users can show or hide the sidebar at will and still navigate effectively with the command palette alone.
- [x] The shell owns pane composition, overlays, responsive breakpoints, live activity infrastructure, and status surfaces.
- [x] Users can manage and switch among multiple canonical sessions efficiently without relying on client-local tab identity.
- [x] `workspace` and `workbench layout` remain clearly separate concepts in both implementation and UX.
- [x] Live activity and Action Center surfaces are driven by typed broker-visible events and objects, not log scraping.
- [x] Users can open, inspect, jump, and back through linked objects consistently across sessions, runs, approvals, artifacts, and audit records.
- [x] Long-form content supports both ordinary terminal selection and explicit copy actions.
- [x] Compose input supports multiline paste cleanly.
- [x] A small running animation indicator appears when canonical work is actively progressing, and shell sync degradation is distinguishable from ordinary loading.
- [x] Persisted layout, theme, recents, pinned sessions, and last-session convenience state remain local-only and non-authoritative.
- [x] The workbench stays topology-neutral for future remote or scaled backends.
