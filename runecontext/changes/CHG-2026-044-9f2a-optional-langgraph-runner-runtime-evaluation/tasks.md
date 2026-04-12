# Tasks

## Adoption Gate

- [ ] Reassess the runner after `CHG-2026-033-6e7b-workflow-runner-durable-state-v0` native hardening is complete.
- [ ] Decide whether LangGraph is still needed for runner-local checkpoint/wait/resume complexity.
- [ ] Record the outcome explicitly: adopt behind the runtime seam or do not adopt.

## Runtime Seam Fit

- [ ] Confirm the runner runtime seam is narrow enough to keep LangGraph fully internal.
- [ ] Ensure LangGraph can be substituted without changing broker local API contracts, protocol schemas, or broker-owned lifecycle/approval semantics.

## Replay + Recovery Evaluation

- [ ] Evaluate LangGraph checkpoint and interrupt behavior against RuneCode replay and idempotency requirements.
- [ ] Confirm plan-bound fail-closed recovery still rejects stale or superseded plan bindings.
- [ ] Confirm restart-safe resume still requires broker validation of approval state, bound scope/hash, and active plan identity.

## Optional Prototype

- [ ] If justified, build a narrow prototype limited to internal checkpoint/wait/resume mechanics.
- [ ] Keep LangGraph checkpoints and thread state non-canonical.
- [ ] Keep runner journal/snapshot, broker reports, and broker-wins reconciliation authoritative.

## Acceptance Criteria

- [ ] LangGraph is implemented only if it remains optional, internal-only, and clearly beneficial.
- [ ] Adoption, if chosen, does not change trust-boundary ownership, broker authority, or public contracts.
- [ ] Replay, wait/resume, and restart semantics remain fail-closed and plan-bound.
