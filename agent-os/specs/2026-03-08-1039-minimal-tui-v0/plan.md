# Minimal TUI v0

User-visible outcome: users can view runs, review diffs/artifacts, approve or deny high-risk actions, and inspect the audit timeline from a local TUI.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-minimal-tui-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Bubble Tea App Skeleton

- Implement the TUI using Bubble Tea.
- Define a simple navigation model (list -> detail, tabbed panes, or routed views).

## Task 3: Core Screens (MVP)

- Runs list + run detail.
- Approvals inbox (manifest signing, container opt-in, other gated actions).
- Artifacts browser (diffs, logs, gate results) with metadata.
- Audit timeline (paged view + verify status).

## Task 4: Local API Integration

- Connect only via the local broker API.
- Use OS peer auth where available.

## Task 5: Safety UX

- Make the active isolation backend and assurance level unmissable.
- Make container mode clearly labeled as reduced assurance.

## Acceptance Criteria

- A user can complete an end-to-end run using the TUI for approvals.
- Diffs/artifacts/audit events are navigable without exposing raw secrets.
