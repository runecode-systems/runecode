# Tasks

## Bubble Tea Shell Foundation

- [x] Implement the TUI using Bubble Tea.
- [x] Follow Bubble Tea's message-driven architecture under the covers:
  - keep `Update` and `View` fast
  - run I/O and long work through commands
  - avoid out-of-band model mutation from goroutines
- [x] Implement a root shell model plus child route/component models rather than a monolithic screen model.
- [x] Define a visible primary navigation model for wide terminals.
- [x] Add a command palette or quick-jump surface for route switching and power-user navigation.
- [x] Add shared generated help output from the real keymap definitions.
- [x] Define a clear focus model and ensure focus state is always visible.
- [x] Support keyboard-only full operation.
- [x] Support mouse as an additive interaction path:
  - click-to-focus
  - click-to-open
  - wheel scrolling
- [x] Ensure every mouse action in MVP has a keyboard-equivalent action.

Parallelization: shell foundation can proceed in parallel with read-model and API work, but it depends on stable route identities and typed data contracts.

## Hybrid MVP Routes

- [x] Implement `Dashboard` as the default landing route.
- [x] Implement `Chat` as a first-class route in the same shell, not a hidden or secondary workflow.
- [x] Implement `Runs` route with run list + run detail.
- [x] Implement `Approvals` route as the MVP action center slice.
- [x] Implement `Artifacts` route with typed artifact browsing and drill-down.
- [x] Implement `Audit` route with timeline, verification posture, and drill-down entry points.
- [x] Implement `Status` route with broker/subsystem readiness and version posture.
- [x] Prefer routed views and inspector panes over modal-heavy drill-down patterns.

Parallelization: route work can be parallelized once shared shell, route registry, and read-model contracts are frozen.

## Chat And Session Foundation

- [x] Build the chat route on top of a minimal canonical session/transcript substrate rather than client-local-only transcript state.
- [x] Support stable session identity for the active MVP session.
- [x] Support ordered transcript turns/messages.
- [x] Support typed send-message request/response or equivalent broker-mediated session interaction.
- [x] Surface linked run, approval, artifact, and audit references from the chat route where those relationships exist.
- [ ] Keep full multi-session browsing, saved workspaces, and session switching out of this change and in the follow-on TUI change.

Parallelization: depends on minimal session/transcript model alignment; rendering and interaction work can proceed in parallel once that contract is stable.

## Runs, Approvals, And Artifacts

- [x] Runs views must consume broker `RunSummary` and `RunDetail` contracts rather than ad hoc screen-specific shaping.
- [x] Run views must distinguish and explain:
  - `backend_kind`
  - runtime isolation assurance
  - provisioning/binding posture
  - audit posture
- [x] Run detail views must surface authoritative vs advisory state explicitly.
- [x] Run detail views must explain partial blocking and coordination waits from coordination/stage/role summaries rather than from a second lifecycle vocabulary.
- [x] Approval views must consume broker approval summaries and detail surfaces rather than payload scraping or local heuristics.
- [x] Approval views must distinguish exact-action approvals from stage sign-off approvals.
- [x] Approval views must surface stale, superseded, expired, consumed, approved, and denied states clearly.
- [x] Approval views must keep `policy_reason_code`, `approval_trigger_code`, and execution/system errors visually and semantically distinct.
- [x] Approval views must show concise structured “what changes if approved” information.
- [x] Approval views must show the canonical bound identity and exact bound scope without exposing daemon-private internals.
- [x] Artifact views must consume broker `ArtifactSummary` and typed read streams rather than daemon-private storage metadata.
- [x] Artifact/detail views must support inspectable diff/log/result content without promoting raw logs to the primary source of control-plane truth.

Parallelization: runs, approvals, and artifacts can be developed in parallel after shared data models and key interaction patterns are defined.

## Audit And Status

- [x] Audit route must provide a paged audit timeline.
- [x] Audit route must surface anchored vs unanchored posture and invalid/failed anchoring states.
- [x] Audit route must consume machine-readable audit verification findings and reason codes rather than scraped prose.
- [x] Audit drill-down must be planned or implemented through typed broker-owned record detail reads rather than direct ledger access.
- [x] Status and diagnostics views must consume broker `BrokerReadiness` and `BrokerVersionInfo` contracts.
- [x] Status views must explain degraded subsystem posture without collapsing everything into one generic unhealthy label.

Parallelization: audit and status surfaces can proceed in parallel with broker read-model work once timeline, verification, and status contracts are stable.

## Live Activity Foundation

- [x] Keep live-update UX grounded in typed watch/event surfaces rather than log scraping.
- [x] Ensure the TUI foundation is ready to consume typed live activity for runs, approvals, and sessions.
- [x] Prefer explicit watch/event families such as run, approval, and session watch streams over one ambiguous event bus.
- [x] Use logs as a supplementary inspection surface rather than the sole live operator surface.

Parallelization: depends on broker/API stream-family alignment; shell and route UI can prepare for these surfaces in parallel.

## Visual System And Theming Foundation

- [x] Use a semantic theme-token layer rather than hard-coded per-view colors.
- [x] Make the TUI colorful, professional, and dense enough for efficient use.
- [x] Ensure color is never the only cue for posture or state.
- [x] Use compact tables, lists, badges, summaries, and inspectors rather than oversized card layouts.
- [x] Support multiple content presentation modes where useful:
  - rendered
  - raw
  - structured
- [x] Preserve a theme foundation that can later support user-selectable presets and customization.

Parallelization: can proceed in parallel with route implementation once semantic state taxonomy and shell layout are defined.

## Local API Integration And Trust Boundaries

- [x] Connect only via the local broker API.
- [x] Use OS peer auth where available.
- [x] Do not scrape broker or daemon CLI output for operational state that already has a typed contract.
- [x] Do not use daemon-private file paths, storage layouts, or local counters as part of the user-facing model.
- [x] Do not invent TUI-local approval or workflow semantics to smooth over missing broker contracts; instead, capture those as control-plane follow-ups.

Parallelization: can proceed in parallel with broker development once transport/auth and logical contracts are specified.

## Safety UX

- [x] Make the active `backend_kind`, runtime isolation assurance, provisioning posture, audit posture, and approval profile unmissable.
- [x] Make container mode clearly labeled as reduced runtime isolation assurance.
- [x] Surface degraded posture states prominently, including:
  - TOFU isolate key provisioning
  - degraded or unavailable authoritative runtime posture
  - unanchored or degraded audit posture
- [x] Keep reduced backend assurance, degraded provisioning posture, degraded audit posture, advisory state, and blocking state visually distinct.
- [x] Keep gate failure, gate override, approval-required, and system-failure states visually distinct rather than flattening them into one label.

Parallelization: can be implemented in parallel with policy, audit, and runtime posture model work once the shared state taxonomy is frozen.

## Acceptance Criteria

- [x] A user can enter the TUI, land on the dashboard, and navigate to the chat route without relying on hidden primary navigation.
- [x] A user can complete an end-to-end run using the TUI for approvals over the real broker local API.
- [x] Runs, approvals, artifacts, audit, and status are inspectable through typed broker contracts rather than CLI scraping or daemon-private metadata.
- [x] The MVP TUI distinguishes authoritative broker state from advisory runner state.
- [x] The MVP TUI distinguishes backend kind, runtime isolation assurance, provisioning posture, and audit posture.
- [x] The MVP TUI is fully usable with keyboard only and offers additive mouse support.
- [x] Diffs, artifacts, and audit events are navigable without exposing raw secrets.
