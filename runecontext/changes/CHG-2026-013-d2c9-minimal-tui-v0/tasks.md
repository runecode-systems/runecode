# Tasks

## Bubble Tea App Skeleton

- [ ] Implement the TUI using Bubble Tea.
- [ ] Define a simple navigation model (list -> detail, tabbed panes, or routed views).

Parallelization: can be implemented in parallel with broker local API work; it depends on a stable logical broker API object model plus the local transport/auth scheme.

## Core Screens (MVP)

- [ ] Runs list + run detail.
- [ ] Approvals inbox (manifest signing, container opt-in, other gated actions).
- [ ] Artifacts browser (diffs, logs, gate results) with metadata.
- [ ] Audit timeline (paged view + verify status).
- [ ] Audit timeline must surface anchored vs unanchored verification posture (and any invalid/failed anchoring state).
- [ ] Audit timeline and posture views consume machine-readable audit verification findings/reason codes from the local API rather than scraping human CLI output.
- [ ] Approval context: show the active approval profile (`moderate` in MVP) and why each approval is required (reason codes + structured details).
- [ ] Distinguish exact-action approvals from stage sign-off approvals in inbox and detail views.
- [ ] Surface when a stage sign-off became stale or was superseded because its bound stage summary hash changed.
- [ ] Runs views consume broker `RunSummary` and `RunDetail` contracts rather than ad hoc screen-specific data shaping.
- [ ] Runs views distinguish and explain:
  - `backend_kind`
  - runtime isolation assurance
  - provisioning/binding posture
  - audit posture
- [ ] Approval views consume broker `ApprovalSummary` and bound-scope metadata so blocked work, supersession, expiry, and consumption are explainable without payload scraping.
- [ ] Keep approval views clear about the difference between `policy_reason_code`, `approval_trigger_code`, and execution/system error states.
- [ ] Artifact views consume broker `ArtifactSummary` plus streamed artifact reads rather than daemon-private storage metadata.
- [ ] Status and diagnostics views consume broker `BrokerReadiness` and `BrokerVersionInfo` contracts.
- [ ] Run detail views surface authoritative vs advisory state explicitly and explain partial blocking from coordination/stage/role summaries rather than from a separate lifecycle label.
- [ ] Gate views surface gate attempts, gate evidence, gate outcomes, and override linkage from typed broker-visible data rather than log scraping.
- [ ] Approval and gate views surface stable bound identities for `run_id`, `stage_id`, `step_id`, `role_instance_id`, and gate attempts where relevant.

Parallelization: screens can be built in parallel, but all depend on the broker local API schemas and shared error taxonomy.

## Local API Integration

- [ ] Connect only via the local broker API.
- [ ] Use OS peer auth where available.
- [ ] Do not scrape broker or daemon CLI output for operational state that already has a typed local-API contract.

Parallelization: can be implemented in parallel with broker development once the local IPC endpoint and auth handshake are specified.

## Safety UX

- [ ] Make the active `backend_kind` and runtime isolation assurance unmissable.
- [ ] Make container mode clearly labeled as reduced runtime isolation assurance.
- [ ] Make the active approval profile unmissable and keep the default posture obvious (`moderate` in MVP).
- [ ] Surface degraded posture states prominently:
  - TOFU isolate key provisioning
  - unanchored audit segments (when anchoring is configured/expected)
- [ ] Keep reduced backend assurance, degraded provisioning posture, and degraded audit posture visually distinct in run and status views.
- [ ] For each approval request, show a concise, structured what changes if approved view.
- [ ] For each approval request, show the canonical bound identity the user is acting on (action-derived request or stage-summary sign-off) without exposing daemon-private internals.
- [ ] Keep authoritative broker state and advisory runner state visually distinct in run and diagnostics views.
- [ ] Keep gate failure, gate override, and approval-required states visually distinct rather than flattening them into one generic blocked/error label.

Parallelization: can be implemented in parallel with policy engine approval payload design; it depends on stable reason codes and structured decision details.

## Acceptance Criteria

- [ ] A user can complete an end-to-end run using the TUI for approvals.
- [ ] Diffs/artifacts/audit events are navigable without exposing raw secrets.
