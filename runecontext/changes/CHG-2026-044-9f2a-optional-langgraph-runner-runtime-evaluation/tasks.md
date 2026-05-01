# Tasks

## Adoption Gate

- [ ] Reassess the runner after `CHG-2026-033-6e7b-workflow-runner-durable-state-v0` native hardening is complete.
- [ ] Decide whether LangGraph is still needed for runner-local checkpoint/wait/resume complexity.
- [ ] Record the outcome explicitly: adopt behind the runtime seam or do not adopt.
- [ ] Require the adoption decision to account for exact-action wait support for hard-floor approvals such as `git_remote_ops`.
- [ ] Require the adoption decision to account for exact-action wait and deferred execution support for external audit anchor submission once that lane is active.

## Runtime Seam Fit

- [ ] Confirm the runner runtime seam is narrow enough to keep LangGraph fully internal.
- [ ] Ensure LangGraph can be substituted without changing broker local API contracts, protocol schemas, or broker-owned lifecycle/approval semantics.
- [ ] Ensure LangGraph does not require relaxing exact-action approval or remote-drift semantics for `git_remote_ops` or similar hard-floor remote-state-mutation lanes.
- [ ] Ensure LangGraph does not require relaxing exact-action approval, target-binding, or deferred prepared and execute semantics for external audit anchor submission or similar hard-floor remote-state-mutation lanes.
- [ ] Ensure LangGraph does not require relaxing validated project-substrate snapshot binding or repository substrate drift handling for project-context-sensitive waits and resumes.
- [ ] Ensure LangGraph can preserve distinct `waiting_operator_input` and `waiting_approval` semantics.
- [ ] Ensure LangGraph can represent multiple simultaneous scoped waits without turning them into one whole-run paused flag.
- [ ] Ensure LangGraph can preserve dependency-aware partial blocking so unrelated eligible work may continue when shared plan, policy, and coordination state allow it.

## Replay + Recovery Evaluation

- [ ] Evaluate LangGraph checkpoint and interrupt behavior against RuneCode replay and idempotency requirements.
- [ ] Confirm plan-bound fail-closed recovery still rejects stale or superseded plan bindings.
- [ ] Confirm restart-safe resume still requires broker validation of approval state, bound scope/hash, and active plan identity.
- [ ] Confirm restart-safe resume preserves `git_remote_ops` hash-bound waits, including relevant artifact hashes and expected result tree identity, and fails closed when those bindings drift.
- [ ] Confirm restart-safe resume preserves external audit anchor hash-bound and target-bound waits, including typed request hash, canonical target descriptor identity, targeted seal binding, and authoritative deferred-state identity, and fails closed when those bindings drift.
- [ ] Confirm restart-safe resume preserves validated project-substrate snapshot binding where project context matters and fails closed when repository substrate posture or bindings drift incompatibly.

## Optional Prototype

- [ ] If justified, build a narrow prototype limited to internal checkpoint/wait/resume mechanics.
- [ ] Keep LangGraph checkpoints and thread state non-canonical.
- [ ] Keep runner journal/snapshot, broker reports, and broker-wins reconciliation authoritative.

## Acceptance Criteria

- [ ] LangGraph is implemented only if it remains optional, internal-only, and clearly beneficial.
- [ ] Adoption, if chosen, does not change trust-boundary ownership, broker authority, or public contracts.
- [ ] Replay, wait/resume, and restart semantics remain fail-closed and plan-bound.
- [ ] Adoption, if chosen, does not weaken exact-action approval or fail-closed remote-drift handling for `git_remote_ops` or similar hard-floor remote-state-mutation lanes.
- [ ] Adoption, if chosen, does not weaken exact-action approval, target binding, deferred execution semantics, or fail-closed drift handling for external audit anchor submission or similar hard-floor remote-state-mutation lanes.
- [ ] Adoption, if chosen, does not weaken validated project-substrate snapshot binding or fail-closed repository substrate drift handling for project-context-sensitive execution.
- [ ] Adoption, if chosen, preserves separate operator-input versus formal-approval waits and scoped blocking semantics.
