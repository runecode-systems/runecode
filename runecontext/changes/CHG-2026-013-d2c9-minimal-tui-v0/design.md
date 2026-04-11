# Design

## Overview
Implement the first RuneCode TUI as a hybrid local terminal client: a dashboard-first operations shell with a first-class chat/coding route, backed entirely by typed broker local API contracts and strict trust-boundary discipline.

This change defines the MVP TUI foundation. It is intentionally broad enough to avoid repainting the substrate later, but it keeps advanced multi-session workbench behavior and deeper observability enhancements in a dedicated pre-MVP follow-on change.

## Product Shape

### Hybrid Shell
- The TUI starts in a dashboard/ops-console route.
- The dashboard is not the only product identity; it is the default landing route for a broader shell.
- The shell also includes a first-class chat/coding route so users can work in RuneCode through an interactive terminal experience rather than only reviewing runs after the fact.
- Runs, approvals, artifacts, audit, and status remain first-class routes or inspectors inside the same shell rather than separate products.

### MVP Intent
- The MVP should feel like a serious terminal client, not a temporary admin panel.
- The MVP should preserve RuneCode’s strict architecture rather than creating a special-case interactive mode.
- The MVP should define a strong foundation for later full multi-session, power-user, and richer observability work.

## Key Decisions
- TUI is a separate least-privilege client; it does not embed privileged execution.
- Use Bubble Tea as the required TUI framework.
- Bubble Tea implementation should follow the framework’s intended message-driven architecture: fast `Update` and `View`, asynchronous I/O in commands, and clear model boundaries.
- The preferred TUI architecture is a root shell model plus child route/component models rather than one monolithic screen model.
- The TUI must consume broker logical API object families directly and must not depend on daemon-private structs, daemon-private storage, local filesystem details, or scraped CLI output.
- The TUI must remain a strict client of the brokered local API and must never invent local authorization, workflow, approval, or audit truth.
- MVP remains local-first, but all boundary-visible TUI assumptions must stay topology-neutral so later remote or scaled backends can present the same logical experience.
- The TUI must preserve the broker distinction between authoritative broker-derived state and runner advisory state.
- The TUI must present backend/runtime posture as separate dimensions rather than flattening them into one overloaded label:
  - `backend_kind`
  - runtime isolation assurance
  - provisioning/binding posture
  - audit posture
- The active approval profile is part of the user safety posture and should be visible and explained (MVP default: `moderate`).
- Approval UI must distinguish exact-action approvals from stage sign-off so it can explain what exact hash-bound work is blocked, what became stale, and what will be unblocked if approval is granted.
- Explanation surfaces must keep `policy_reason_code`, `approval_trigger_code`, and system errors distinct rather than flattening them into one generic status string.
- Container reduced-assurance posture, TOFU-only provisioning posture, and degraded audit posture must remain visually distinct.
- The TUI should explain partial blocking and coordination waits from `RunDetail` coordination/stage/role surfaces rather than inventing a second lifecycle vocabulary.
- Gate attempts, gate evidence, and gate overrides should surface through typed shared contracts rather than log-only heuristics.
- The TUI should surface canonical bound identities for run/stage/step/role scopes and gate attempts without promoting daemon-private identifiers to user authority.
- Raw model chain-of-thought is out of scope for MVP. Inspectability should focus on typed traces, tool activity, approvals, policy decisions, artifacts, audit-linked events, and rationale summaries where those exist.

## Interaction Model

### Navigation Principles
- Keyboard-only operation must be fully supported.
- Mouse support must be additive, not required.
- Every mouse-triggered action in MVP must have a keyboard-equivalent path.
- Primary navigation should be visible on wide layouts rather than hidden behind a hamburger-first shell.
- On narrower layouts, the TUI may compact navigation, but route-switching must remain discoverable through visible affordances plus keyboard commands.
- A command palette or quick-jump surface is part of the MVP foundation for efficient navigation.
- Global shortcuts should be consistent across routes.
- Contextual shortcut help should be visible and generated from the real keymap definitions rather than hand-maintained text.

### Focus And Drill-Down
- The TUI must always make focus state clear.
- The shell model owns global keys, global route switching, and shared status/help surfaces.
- Active route models own route-local interaction and rendering.
- Drill-down patterns should prefer `summary -> detail pane` or `summary -> routed detail view` over modal-heavy workflows.
- Drawers and overlays should be reserved for small contextual actions or confirmations, not for long-form inspection flows.

### Default Navigation Setup To Prefer
- Default route: `Dashboard`.
- First-class routes in MVP:
  - `Dashboard`
  - `Chat`
  - `Runs`
  - `Approvals`
  - `Artifacts`
  - `Audit`
  - `Status`
- The shell should also support an `Action Center` concept, but MVP may realize that as the `Approvals` route until a separate pending-question model exists.

## Visual Language

### Design Goals
- The TUI should be colorful, streamlined, and professional.
- Color is both aesthetic and semantic.
- Color must never be the only cue.
- The interface should be dense enough that users do not have to work hard to get the information they need, but it must still feel polished and readable.

### Visual Rules
- Use semantic theme tokens rather than hard-coded per-screen colors.
- Semantic states such as active, pending, blocked, degraded, failed, approved, denied, stale, and advisory should map to consistent styling across routes.
- Degraded security posture must remain distinct even in limited-color or no-color terminals.
- Wide-screen layouts should prefer visible information scent and stable navigation over hidden chrome.
- Oversized marketing-style cards and large empty padding should be avoided in favor of compact tables, lists, badges, summaries, and inspectors.
- Long-form content such as markdown, logs, diffs, and structured objects should support multiple display modes where useful:
  - rendered
  - raw
  - structured
- Theme customization is a later feature, but the MVP implementation must preserve a theme-token foundation so user-selectable presets can be added without rewriting views.

## Data Model Expectations

### Read Models The TUI Depends On
The MVP TUI should be built around first-class broker read models rather than screen-local data shaping:
- `RunSummary`
- `RunDetail`
- `ApprovalSummary`
- `ArtifactSummary`
- `AuditTimelinePage`
- `AuditVerificationReport`
- `BrokerReadiness`
- `BrokerVersionInfo`

### Approval Read Model Expectations
- MVP screens may begin from `ApprovalSummary`, but the overall TUI foundation requires a richer approval-detail surface than one prose field and optional raw envelopes.
- The broker/API lane should add or plan a richer approval-detail contract that surfaces:
  - `policy_reason_code`
  - binding kind (`exact_action` vs `stage_sign_off`)
  - structured explanation of what changes if approved
  - blocked-work scope
  - stale/superseded/expired semantics with typed reason codes
- The TUI should not derive these by scraping or deeply inspecting signed payloads in the client.

### Audit Read Model Expectations
- Audit timeline and verification reads are foundational but not sufficient for deep inspection.
- The TUI foundation assumes the broker/API lane will support audit record drill-down through typed broker-owned reads rather than daemon-private ledger access.
- Timeline entries should carry stable canonical record identities so users can pivot into record detail and related references.

### Minimal Session Foundation
- Because the MVP includes a first-class chat route, the TUI foundation assumes a minimal canonical session/transcript model rather than client-local-only transcript state.
- The minimum useful session substrate should support:
  - stable session identity
  - ordered turns/messages
  - send-message request/response semantics
  - links from turns to runs, approvals, artifacts, and audit references where relevant
- Full multi-session management is deferred, but the MVP must not paint itself into a single-session client-only corner.

## Live Activity Model

### Principle
- Live UX should come from typed watch/event contracts, not from log scraping and not from a second local state authority.

### MVP Foundation Expectation
- The stream model should be ready for additive typed watch families beyond logs and artifact reads.
- The most important additions for the TUI foundation are:
  - `RunWatchEvent`
  - `ApprovalWatchEvent`
  - `SessionWatchEvent`
- These event families should make it possible to surface:
  - run lifecycle and progress changes
  - blocking and resume posture
  - approval creation, resolution, expiry, and supersession
  - chat/session turn streaming and tool activity summaries
- The TUI may still expose logs, but logs are not the primary source of live control-plane truth.

## Auditability And Inspectability
- The TUI should make typed system behavior inspectable without elevating raw internal reasoning to a trust primitive.
- MVP inspectability should focus on:
  - chat/session turns
  - tool activity summaries
  - approvals and decisions
  - run/stage/role/gate progress
  - artifacts and diffs
  - audit timeline and audit verification posture
  - linked rationale or explanation summaries where those are defined
- The TUI should preserve the distinction between authoritative and advisory views throughout inspection flows.

## Safety UX
- Make the active `backend_kind`, runtime isolation assurance, provisioning posture, audit posture, and approval profile unmissable.
- Make reduced-assurance and degraded states distinct from each other.
- Keep authoritative broker state and advisory runner state visibly distinct.
- Keep gate failure, gate override, approval-required, and system-failure states distinct rather than flattening them into one generic blocked/error label.
- Show concise structured “what changes if approved” summaries for approval actions.
- Show canonical bound identity and exact bound scope for approvals without exposing daemon-private internals.

## Foundation Shortcuts To Avoid
- Do not turn the dashboard into a hidden-nav launcher for the rest of the TUI.
- Do not model chat history or live activity as client-local-only state if those concepts need to survive beyond the current terminal process.
- Do not depend on raw logs as the only live-activity or inspection surface.
- Do not collapse approvals and future pending questions into one undifferentiated queue.
- Do not collapse backend, assurance, audit, and coordination posture into one ambiguous status field.
- Do not bypass broker contracts to make the UI feel faster.
- Do not use modal-heavy interaction as the main drill-down strategy.
- Do not promise raw model chain-of-thought capture or display for MVP.

## Main Workstreams
- Bubble Tea Shell Foundation
- Hybrid Dashboard + Chat Routes
- Typed Local API Integration
- Safety and Posture UX
- Approval / Audit / Live Activity Contract Alignment

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, audit, live activity, or typed contracts, the plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
