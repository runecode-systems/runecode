# Tasks

## Workspace Concurrency Model

- [ ] Define the supported concurrency modes explicitly:
  - default single-run-per-workspace mode
  - any later shared-workspace modes
- [ ] Keep fail-closed defaults:
  - one active run per workspace remains the default
  - any non-default concurrency posture requires explicit design and approval
- [ ] Define the lock/lease model for workspace ownership, including acquisition, renewal, expiry, and crash recovery semantics.
- [ ] Key concurrency locks, leases, and conflict scopes off the shared stable logical workflow identities rather than retry/attempt-local IDs.
- [ ] Preserve separate attempt identities for retries and reruns so concurrency logic does not overload logical scope identity.

## Conflict Detection + Isolation Rules

- [ ] Define deterministic conflict handling for concurrent runs that touch the same logical workspace state.
- [ ] Account for approval binding, artifact routing, gate results, and recovery semantics under concurrency.
- [ ] Require explicit policy and audit recording when a run uses any non-default concurrency posture.
- [ ] Ensure approvals remain run-bound and cannot be consumed across runs under shared-workspace modes.
- [ ] Ensure gate attempts, gate evidence, and gate overrides remain bound to the originating run/stage/step scope under concurrency.
- [ ] Account for validated project-substrate snapshot identity and project-substrate drift under concurrency.
- [ ] Ensure one run cannot silently satisfy or inherit another run's project-context preconditions or remediation posture.

## Runner, Broker, and TUI Integration

- [ ] Define how the runner, broker, and local API expose concurrency posture, lock ownership, waits, and conflicts.
- [ ] Keep TUI and CLI surfaces clear when a run is blocked by another active run or is sharing a workspace under an explicit concurrency mode.
- [ ] Record lock acquisition, release, contention, overrides, and conflict-triggered failures as audit events.
- [ ] Reuse or extend the shared broker run-detail and coordination-summary contracts rather than inventing a separate concurrency-specific UI status model.
- [ ] Keep public run lifecycle on the shared broker lifecycle vocabulary; represent lock waits and partial blocking through coordination/detail surfaces rather than a new lifecycle enum.
- [ ] Surface differing validated project-substrate snapshot bindings explicitly when concurrent runs depend on project context.

## Fixtures + Recovery Cases

- [ ] Add checked-in fixtures and test cases for:
  - normal lock acquisition/release
  - contention
  - expired locks
  - crash recovery
  - conflict-triggered failures

## Acceptance Criteria

- [ ] Default behavior remains one active run per workspace.
- [ ] Concurrent use of one workspace requires an explicit design and fail-closed posture.
- [ ] Locking, contention, and recovery are auditable and deterministic.
- [ ] Approval, artifact, and gate semantics stay bound to the correct run under concurrency.
- [ ] Concurrency integrates with the shared identity, lifecycle, approval, gate-evidence, and validated project-substrate binding model without introducing parallel workflow semantics.
