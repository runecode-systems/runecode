# Minimal TUI v0

User-visible outcome: users can view runs, review diffs/artifacts, approve or deny high-risk actions, and inspect the audit timeline from a local TUI.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-minimal-tui-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Bubble Tea App Skeleton

- Implement the TUI using Bubble Tea.
- Define a simple navigation model (list -> detail, tabbed panes, or routed views).

Parallelization: can be implemented in parallel with broker local API work; it depends on a stable local API transport/auth scheme.

## Task 3: Core Screens (MVP)

- Runs list + run detail.
- Approvals inbox (manifest signing, container opt-in, other gated actions).
- Artifacts browser (diffs, logs, gate results) with metadata.
- Audit timeline (paged view + verify status).
- Audit timeline must surface anchored vs unanchored verification posture (and any invalid/failed anchoring state).
- Approval context: show the active approval profile (`moderate` in MVP) and why each approval is required (reason codes + structured details).

Parallelization: screens can be built in parallel, but all depend on the broker local API schemas and shared error taxonomy.

## Task 4: Local API Integration

- Connect only via the local broker API.
- Use OS peer auth where available.

Parallelization: can be implemented in parallel with broker development once the local IPC endpoint and auth handshake are specified.

## Task 5: Safety UX

- Make the active isolation backend and assurance level unmissable.
- Make container mode clearly labeled as reduced assurance.
- Make the active approval profile unmissable and keep the default posture obvious ("moderate" in MVP).
- Surface degraded posture states prominently:
  - TOFU isolate key provisioning
  - unanchored audit segments (when anchoring is configured/expected)
  - untested-but-probe-passing vendor bridge runtimes (post-MVP)
- For each approval request, show a concise, structured "what changes if approved" view.

Parallelization: can be implemented in parallel with policy engine approval payload design; it depends on stable reason codes and structured decision details.

## Acceptance Criteria

- A user can complete an end-to-end run using the TUI for approvals.
- Diffs/artifacts/audit events are navigable without exposing raw secrets.
