# Workflow Runner + Workspace Roles + Deterministic Gates v0

User-visible outcome: RuneCode can execute an end-to-end run where the scheduler proposes steps, policy authorizes them, workspace roles perform work offline, and deterministic gates produce evidence artifacts.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-workflow-workspace-roles-gates-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Workflow Runner Contract (Untrusted Scheduler)

- Implement a TS/Node workflow runner using LangGraph.
- Ensure the runner has no direct secrets and no direct workspace access.
- All actions are requested through the broker/local API and independently validated by the launcher/policy engine.
- Persist run state durably so pause/resume and crash recovery are real (MVP: SQLite):
  - run state machine (proposed/validated/authorized/executing/awaiting_approval/failed/succeeded)
  - step attempts, artifact references, and approval records
  - idempotency/replay rules for retrying after crashes
- Define MVP concurrency rules:
  - default: one active run per workspace (explicit workspace lock)
  - concurrent runs require explicit design and are post-MVP unless proven safe

## Task 3: Workspace Roles (MVP Set)

- Define and implement the MVP workspace roles:
  - `workspace-read` (RO)
  - `workspace-edit` (RW, offline)
  - `workspace-test` (snapshot + discard)
- Ensure command execution is via purpose-built executors/allowlists (no shell passthrough).

## Task 4: Propose -> Validate -> Authorize -> Execute -> Attest Loop

- Treat model output as untrusted proposals.
- Validate proposals structurally (schema, size, artifact references).
- Authorize deterministically via policy engine.
- Execute inside the correct role isolate.
- Attest by producing signed artifacts (diffs, logs, gate results) and audit events.

## Task 5: Deterministic Gates (MVP)

- Implement a gate framework with evidence artifacts for:
  - build/type checks
  - tests
  - lint/format
  - secret scanning
  - policy compliance checks
- Define gate failure semantics:
  - default: gate failure fails the step/run deterministically
  - retries are explicit and recorded
  - any override requires a recorded human approval and produces an audit event

## Task 6: Minimal End-to-End Demo Run

- Provide a single “demo workflow” that runs on Linux:
  - creates a small change in a demo workspace
  - runs gates
  - produces audit + artifacts
  - requires at least one explicit approval (e.g., manifest sign-off)

## Acceptance Criteria

- A run can be started, paused for approval, resumed, and completed.
- A run can be recovered after a crash/restart of the scheduler process (no "in-memory" state required to resume).
- Gates are deterministic and produce verifiable artifacts.
- The scheduler cannot exceed policy or bypass gates.
