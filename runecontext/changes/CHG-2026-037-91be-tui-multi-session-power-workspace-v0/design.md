# Design

## Overview
This change turns the alpha TUI foundation into a full-screen workbench built on Bubble Tea, Lip Gloss, and selective Bubbles components. It keeps the root shell plus child-model architecture from `CHG-2026-013-d2c9-minimal-tui-v0`, but upgrades the shell into a real pane compositor and shared service layer.

This change does not replace the MVP foundation. It freezes the next layer of product shape and interaction rules on top of that foundation so future features build on one workbench substrate instead of inventing route-local UI behavior.

## Current Gap
- the current shell is still mostly top-level navigation plus route-local linear renderers
- each route largely hand-builds its own list, inspector, loading, and focus treatment
- the shell does not yet own pane composition, overlay stack, global watch infrastructure, copy/paste behavior, or responsive breakpoints
- without a shell substrate upgrade, multi-session, Action Center, and observability work would fragment into route-local UX and duplicated infrastructure

## Goals
- provide a polished non-shell terminal application feel with full terminal takeover, clear pane hierarchy, visible focus states, and strong selection cues
- preserve discoverability while adding power-user speed through toggleable navigation, command surfaces, shortcuts, and denser layouts
- keep canonical broker identities and typed models as the source of truth for sessions, runs, approvals, artifacts, and audit records
- make copy and paste simple for long-form content without sacrificing normal terminal text selection
- preserve a local-first and topology-neutral client posture so future remote or scaled backends can reuse the same logical workbench behavior

## Non-Goals
- replacing Bubble Tea or Lip Gloss with another framework
- collapsing the shell plus child-model architecture back into one monolithic model
- rewriting RuneCode's workbench around `crush`'s screen-buffer and rectangle-draw architecture
- treating the `crush` codebase as a template to copy wholesale rather than a source of bounded-rendering and component-discipline patterns
- inventing pending-question or answer-required semantics before a canonical broker model exists
- persisting local layout or theme state as authority
- turning the TUI into a log console that infers truth from daemon output
- requiring multiple equal session panes in the first advanced workbench cut

## Locked Decisions
- Bubble Tea remains the required framework and architectural posture.
- Lip Gloss remains the required layout and styling system.
- Bubbles may be used selectively for `viewport`, `textarea`, `spinner`, and shared help primitives, but not in a way that forces generic list or table UX where custom models are a better fit.
- The root shell plus child-model architecture continues, but the shell now owns full terminal layout, overlays, breakpoint behavior, and shell-level status surfaces.
- Default launch posture is full-screen alt-screen mode.
- Primary navigation is an optional left sidebar plus an object-aware command palette.
- The sidebar is visible by default, toggleable, and optional; palette-only navigation must remain fully capable when the sidebar is hidden.
- One active session in the main pane is the default session model. Multiple equal session panes are future work for wider terminals, not the first foundation requirement.
- `workspace` means canonical broker workspace identity.
- `workbench layout` means local saved pane, sidebar, and inspector arrangement.
- Persisted workbench state is local convenience state only.
- Long-lived watch streams and global object-summary caches are shell-owned workbench infrastructure, not per-route state.
- Action Center v0 families are approvals, operational attention, and blocked work impact. Future questions remain reserved until a canonical broker model exists.
- Copy and paste must support both terminal text selection and explicit in-app copy actions; terminal selection must never be sacrificed.
- The shell must show a small running animation indicator when canonical work is actively progressing.
- The larger route-level visual pass is downstream of workflow semantic stabilization and first-round dogfooding.

## External Reference Review: Charmbracelet Crush

The project reviewed Charmbracelet `crush` directly as a current Go, Bubble Tea, and Lip Gloss reference implementation.

Key conclusions:
- `crush` uses the same core ecosystem, but not the same root architecture.
- `crush` is more component-oriented and rectangle-bounded, with component-local draw ownership and a screen-buffer composition pass.
- RuneCode's TUI is intentionally shell-planner and route-surface oriented, with the shell composing pane contracts returned by routes.
- RuneCode should keep that shell architecture because it matches the product's multi-pane, object-workbench posture and existing trust-boundary-aware shell ownership.

Patterns worth borrowing from `crush`:
- component-local bounded rendering so already-rendered blocks are not repeatedly flattened and re-constrained by higher-level shell code
- reusable list and overlay primitives that own selection, scrolling, gaps, and bounded viewport math locally
- stronger overlay layering and clipping discipline so dialogs and switchers render within explicit bounds instead of relying on incidental whole-screen string composition behavior
- broader layout regression tests around resizing, anchoring, clipping, gap preservation, and narrow or medium breakpoint behavior

Patterns explicitly not adopted from `crush`:
- a root rewrite to a screen-buffer-first draw model
- a chat-first UI decomposition that would weaken RuneCode's route and inspector contracts
- wholesale copying of `crush` package structure or large model files

## Terminology And Identity
- `workspace`: canonical broker or control-plane workspace identity.
- `session`: canonical broker-visible session or transcript identity associated with a workspace.
- `workbench layout`: a local arrangement of sidebar visibility, pane sizes, inspector state, and similar UI concerns.
- `workbench state`: local convenience state such as recents, pinned sessions, theme preset, or last-viewed object cues.
- `object reference`: a UI-level wrapper around canonical IDs for session, run, approval, artifact, audit record, or future queue families. It must never invent authority beyond existing broker identities.
- `activity cache`: an ephemeral, shell-owned presentation cache derived from typed watch streams. It is never authoritative system state.

## Shell Architecture

### Fullscreen Compositor
The shell should own these regions from day one:
- top status bar
- optional left sidebar
- primary main pane
- optional right inspector pane
- bottom composer or status strip
- shell-owned overlay stack

Child routes should contribute content into shell-owned region contracts rather than owning the whole terminal layout themselves.

### Top Status Bar
- show breadcrumbs, current object context, watch-sync posture, running indicator, and compact key hints
- support focus and state awareness without becoming the primary navigation surface
- replace the idea that top-only route text is sufficient for advanced navigation

### Sidebar
- visible by default
- one-key toggle to show or hide
- hosts primary navigation, session directory, or other shell-owned directories depending on context
- when hidden, the command palette must still expose full navigation coverage
- on narrow terminals, degrade to an overlay or drawer instead of remaining permanently on screen

### Right Inspector Pane
- standard place for cross-object inspection on wide layouts
- can be hidden or collapsed
- on narrow layouts, `inspect` degrades to a full-screen inspector view or overlay rather than disappearing

### Bottom Composer Or Status Strip
- hosts chat or session compose input when relevant
- otherwise shows status, hints, selection-mode state, copy affordances, and shell notices
- belongs to the shell rather than each route inventing its own footer behavior

### Overlay Stack
The shell owns overlays so focus, dismissal, and stacking remain consistent:
- command palette
- object switchers
- dialogs and confirmations
- help and key references
- narrow-screen inspector overlays
- narrow-screen sidebar or directory overlays

## Responsive Layout Model
- wide terminals: sidebar plus main pane plus inspector when useful
- medium terminals: main pane plus one secondary pane; sidebar or inspector may collapse
- narrow terminals: single main pane with overlays or sheets for sidebar, inspector, and switchers
- breakpoint behavior is standardized in the shell, not improvised route by route
- resizable pane ratios are local convenience state and must restore cleanly without affecting canonical object state

## Shared Component And Service Layer

### Shared UI Primitives
- selectable directories and lists with bounded rendering ownership rather than shell-level post-processing of already-rendered rows
- long-form scrollable viewports for transcript, diff, log, markdown, and raw structured content
- inspector headers and identity or status badges
- tabs or mode switches for `rendered`, `raw`, and `structured` views
- toasts and status notices
- empty, loading, and error surfaces
- centered overlays and switchers
- help surfaces driven by real key definitions

### Shell-Owned Services
- focus manager
- overlay manager
- command registry
- watch manager
- clipboard service
- workbench state store
- toast or status service

### Implementation Guidance
- use Bubble Tea to keep the root event loop fast and child models composable
- use Lip Gloss semantic tokens for surface hierarchy, borders, focus, selection, and semantic state cues
- use Bubbles selectively where it accelerates the foundation without forcing generic UX
- prefer custom selectors or palette models when stock list or table models do not match the intended workbench behavior
- prefer component-local width and height ownership for lists, overlays, and pane detail blocks so the shell does not reflow or clip already-rendered subcomponents line by line
- add regression tests whenever a layout or overlay bug is fixed so width, clipping, section-gap, and viewport anchoring behavior remain stable across breakpoints

## Navigation And Object Model

### Navigation Posture
- top-level routes remain visible concepts, but the workbench should feel object-centric rather than route-centric
- the command palette should open routes, sessions, runs, approvals, artifacts, audit records, and shell commands such as pane toggles and layout actions
- visible navigation plus palette support must keep the workbench discoverable without forcing users to memorize hidden command names

### Standard Verbs
- `open`: replace the main-pane context with the chosen object
- `inspect`: open the chosen object in the right inspector or narrow-screen inspector overlay
- `jump`: follow a linked canonical reference while preserving backstack or breadcrumb context
- `back`: return within the object navigation stack, not only the current route index

### Backstack And Breadcrumbs
- the shell should preserve enough context for users to jump from session to approval to audit record and back again
- breadcrumbs and back behavior belong in the shell and should not be reinvented per route

## Multi-Session Model

### Session Workspace Default
- the main workbench defaults to one active session in the main pane
- many sessions can be visible through the sidebar or quick switchers
- pinned sessions, recent sessions, and local "new activity since viewed" cues are allowed as local convenience state
- the shell should not block future wide-terminal multi-pane session layouts, but those are not the first foundation requirement

### Background Session Awareness Minimum
Session directories or switchers should show, at minimum:
- canonical session ID or human-facing label when available
- canonical workspace ID
- last activity time
- last activity kind
- short preview text
- incomplete-turn state
- a high-level state cue such as active, waiting, blocked, degraded, idle, or failed
- linked run count
- linked approval count

### Cross-Object Linking
- sessions remain linked to runs, approvals, artifacts, and audit references through canonical broker-visible identifiers
- quick switching must not depend on client-local tab identity

## Global Watch, Live Activity, And Running Semantics

### Shell-Owned Watch Infrastructure
- the shell should own long-lived follow watchers for `RunWatchEvent`, `ApprovalWatchEvent`, and `SessionWatchEvent`
- watch streams feed an ephemeral presentation cache of broker-projected summaries and a derived live activity feed
- this cache exists for UX responsiveness only and is never authority

### Live Activity Surface
- a cross-object stream showing what is active, waiting, blocked, degraded, completed, or failed
- users should not need to open each route and poll manually to understand current system state
- drill-down from a live activity item should use the same `open` and `inspect` semantics as the rest of the workbench

### Shell-Level Running Indicator Semantics
- `loading`: the UI has a local request in flight
- `running`: canonical session or run activity is actively progressing
- `degraded sync`: the watch or update path is unhealthy or disconnected
- the shell shows a small animated indicator in the status bar when work is actively running
- row-level or pane-level indicators may mirror the same source for the active object

## Action Center Model

### In-Scope Queue Families For v0
- approvals
- operational attention
- blocked work impact

### Reserved Queue Families
- future questions or answer-required prompts may appear only after a canonical broker model exists
- the TUI must not invent question semantics locally before that model exists

### Queue Behavior
- keep approvals, operational attention, and blocked-work cues visibly distinct
- surface counts, urgency, expiry, stale or superseded posture, and blocked-work impact
- support fast keyboard triage and quick switching
- do not merge system errors, approval-required work, and passive blocked states into one overloaded status vocabulary

## Inspector And Content Model

### Shared Inspector Shell
Every major object inspector should use the same structural shell:
- summary header with identity and posture badges
- linked references and related objects
- local actions
- presentation-mode switch: `rendered`, `raw`, `structured`
- copy-friendly long-form content region

### Long-Form Content Surfaces
- transcripts, diffs, logs, markdown, and raw structured objects should use stable scrollable viewports
- wrapping and focus behavior should be predictable so copied text is easy to understand and reproduce
- rendered views matter, but raw and structured views remain first-class

### Inspection Coverage
The shared inspector foundation should support:
- session transcripts and linked activity
- runs, stages, roles, and gate or evidence posture
- approvals and resolution history
- artifacts, diffs, logs, and evidence
- audit records and audit verification posture
- policy decisions or rationale summaries where broker-visible models exist

### Audit Drill-Down
- audit inspection continues to rely on broker-owned derived views and typed reads
- the TUI must not read ledger files directly or treat daemon-private storage as a UI API

## Copy, Paste, And Input Model

### Terminal Selection Flow
- terminal text selection must remain a supported first-class flow for transcripts, output, diffs, logs, and inspectors
- the workbench should not rely on obscure terminal-specific shortcuts as the only way to copy text
- mouse drag must not be required for core UI actions that would make normal text selection impractical

### Selection Mode
- the shell should provide a visible selection mode that reduces or disables mouse capture so drag-to-select behaves normally
- selection mode state should be obvious in the status strip and easy to exit

### Explicit Copy Actions
The workbench should also support in-app copy actions such as:
- copy canonical ID
- copy digest
- copy raw block
- copy transcript excerpt
- copy artifact preview
- copy linked reference list

### Clipboard Behavior
- use native clipboard support or OSC52 when available
- never make clipboard integration the only copy path

### Paste Behavior
- the composer should move to a proper text area model with multiline paste and bracketed paste support
- paste should not depend on ad hoc rune accumulation

## Local Persistence Model

### Initial Local-Only Persisted State
- sidebar visible or hidden
- pane ratios and collapsed states
- inspector visibility
- preferred presentation mode
- theme preset
- last active session per canonical workspace
- recent objects
- pinned sessions
- optionally, last-opened primary route if it does not blur canonical state boundaries

### Persistence Rules
- persisted state is keyed by logical broker target plus canonical workspace or session identifiers when relevant
- host-local details such as socket paths must not become semantic identity
- persisted state must never alter approval truth, run truth, queue truth, or any other broker-owned control-plane meaning

## Visual And Theme Model

### Visual Direction
- dark, calm, app-like canvas
- strong focus highlights and selected-row treatment
- clear pane framing and hierarchy
- centered overlays and command surfaces
- dense but restrained spacing
- integrated diff, markdown, and structured viewers
- obvious current activity without turning the UI into a log console

### Inspiration Rule
- the visual direction may take cues from OpenCode-style fullscreen workbench examples reviewed during planning
- RuneCode should not clone another product literally; it should adopt the same level of compositional polish while preserving its own semantics and trust posture

### Theme System
- themes are defined through semantic tokens for surfaces, borders, focus, selection, text tiers, overlays, and semantic states
- a theme preset may change expression, not meaning
- color is never the only state cue

### Deferred Larger Visual Pass
- the shell substrate should already feel polished before the larger visual pass begins
- the larger route-level visual pass happens after dashboard data expectations, Action Center semantics, route inventory, and first-round dogfooding are stable enough to avoid repaint churn

## Clarifications Frozen By This Change
- canonical shell regions and pane roles are shell-owned
- `workspace` and `workbench layout` are separate terms with separate meanings
- background session awareness minimum fields are specified above
- `open`, `inspect`, `jump`, and `back` are standard workbench verbs
- copy and paste expectations require both terminal selection and explicit copy actions
- running indicator semantics distinguish `loading`, `running`, and `degraded sync`
- Action Center families for v0 are approvals, operational attention, and blocked work impact
- initial local persisted state is explicitly limited to convenience data
- long-lived watch streams and object-summary caches are shell-owned infrastructure, not per-route state

## Future Topology Neutrality
- the workbench must continue to avoid local-only semantic assumptions
- session identity, approval identity, audit record identity, and other user-facing control-plane objects must remain topology-neutral
- this change does not add remote transport, but it must preserve the path for a local TUI to later target a vertically or horizontally scaled backend through the same logical contracts

## Foundation Shortcuts To Avoid
- do not implement multi-session as client-local tabs detached from canonical session identity
- do not let child routes continue to own full-terminal layout once the workbench shell exists
- do not make live activity a log viewer dressed up as a control-plane surface
- do not invent question semantics locally before a canonical model exists
- do not let saved layouts, theme state, recents, or last-opened objects become authority inputs
- do not hide primary navigation behind a hamburger-first or palette-only shell
- do not sacrifice ordinary text selection in the name of mouse-driven panes
- do not replace typed inspection with prose-only summaries where canonical structured data exists

## Main Workstreams
- Fullscreen workbench shell substrate
- Shared component and service layer
- Session workspace and object navigation foundation
- Shell-owned watch manager and live activity cache
- Action Center families and triage flows
- Shared inspector and long-form content foundation
- Copy and paste plus input UX
- Local persistence, responsive layouts, and theme presets
- Deferred larger visual pass after workflow stabilization

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches sessions, approvals, audit, observability, topology, or typed contracts, the plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
