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
- [ ] Runs views consume broker `RunSummary` and `RunDetail` contracts rather than ad hoc screen-specific data shaping.
- [ ] Approval views consume broker `ApprovalSummary` and bound-scope metadata so blocked work, supersession, expiry, and consumption are explainable without payload scraping.
- [ ] Artifact views consume broker `ArtifactSummary` plus streamed artifact reads rather than daemon-private storage metadata.
- [ ] Status and diagnostics views consume broker `BrokerReadiness` and `BrokerVersionInfo` contracts.

Parallelization: screens can be built in parallel, but all depend on the broker local API schemas and shared error taxonomy.

## Local API Integration

- [ ] Connect only via the local broker API.
- [ ] Use OS peer auth where available.
- [ ] Do not scrape broker or daemon CLI output for operational state that already has a typed local-API contract.

Parallelization: can be implemented in parallel with broker development once the local IPC endpoint and auth handshake are specified.

## Safety UX

- [ ] Make the active isolation backend and assurance level unmissable.
- [ ] Make container mode clearly labeled as reduced assurance.
- [ ] Make the active approval profile unmissable and keep the default posture obvious (`moderate` in MVP).
- [ ] Surface degraded posture states prominently:
  - TOFU isolate key provisioning
  - unanchored audit segments (when anchoring is configured/expected)
- [ ] For each approval request, show a concise, structured what changes if approved view.

Parallelization: can be implemented in parallel with policy engine approval payload design; it depends on stable reason codes and structured decision details.

## Acceptance Criteria

- [ ] A user can complete an end-to-end run using the TUI for approvals.
- [ ] Diffs/artifacts/audit events are navigable without exposing raw secrets.
