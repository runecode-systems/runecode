# Design

## Overview
Evaluate whether LangGraph provides enough implementation leverage for runner-local checkpointing and wait/resume orchestration to justify adoption as an internal runtime, without altering RuneCode's native plan-bound, broker-authoritative execution model.

## Key Decisions
- LangGraph remains optional; implementation should be decided at delivery time based on whether the native runner foundation still leaves enough orchestration complexity to justify it.
- Native thin-kernel runner hardening remains the prerequisite and baseline.
- Any LangGraph usage must stay behind the internal runtime seam established by `CHG-2026-033-6e7b-workflow-runner-durable-state-v0`.
- LangGraph must remain internal and non-canonical.
- Broker-owned run truth, approval truth, lifecycle state, and immutable `RunPlan` authority remain unchanged.
- LangGraph thread state, checkpoints, and interrupts must not replace runner journal/snapshot records, typed broker reports, or broker-wins reconciliation.
- Adoption should proceed only if it reduces runner-local implementation risk without weakening trust boundaries, auditability, or fail-closed recovery semantics.
- Adoption must preserve exact-action wait semantics for hard-floor approvals such as `git_remote_ops`, including canonical action hashes, relevant artifact hashes, expected result tree identity, and fail-closed remote-drift handling.
- Adoption must preserve the same exact-action wait semantics for external audit anchor submission once that lane is active, including canonical target descriptor identity, typed request hash, targeted seal identity, and fail-closed target-drift or proof-drift handling.
- Adoption must preserve validated project-substrate snapshot binding and fail-closed repository substrate drift handling where waits or resumes depend on project context.
- Adoption must preserve the shared distinction between `waiting_operator_input` and `waiting_approval` rather than collapsing operator guidance into formal approval waits.
- Adoption must preserve multiple simultaneous scoped waits and dependency-aware partial blocking instead of forcing one whole-run paused flag.
- Adoption must preserve deferred prepared and execute semantics for hard-floor remote-state-mutation lanes rather than collapsing durable prepared state into opaque runtime-local interrupt memory.

## Adoption Criteria

LangGraph should be implemented only if all of the following are true at that time:

- the native runner durable-state and approval-wait model is already complete and verified
- the runtime seam is in place and small enough to keep LangGraph fully internal
- LangGraph measurably reduces runner-local orchestration complexity for pause/wait/resume flows
- replay, interrupt, and checkpoint semantics can be bound cleanly to the same `run_id`, `plan_id`, scope identity, attempt identity, and idempotency model RuneCode already uses
- broker validation still remains authoritative for resume and reconciliation
- exact-action waits for hard-floor remote-state-mutation approvals such as `git_remote_ops` remain hash-bound, non-batchable, and fail closed on changed remote or artifact bindings
- exact-action waits for external audit anchor submission remain hash-bound, target-bound, non-batchable, and fail closed on changed typed request, target identity, or proof bindings
- multiple simultaneous scoped waits can be restored without collapsing them into one opaque runtime-global pause state

## Non-Goals

LangGraph adoption must not:

- become a planner or workflow compiler
- become a policy engine or approval authority
- define a second public lifecycle vocabulary
- require broker/API contracts to mirror LangGraph thread/checkpoint vocabulary
- replace explicit runner journal families with opaque framework-owned blobs
- weaken exact-action approval or remote-drift semantics for git remote mutation or other hard-floor remote-state-mutation lanes
- weaken exact-action approval, target binding, or deferred prepared and execute semantics for external audit anchor submission or other hard-floor remote-state-mutation lanes

## Evaluation Areas

### Runtime Fit
- Can LangGraph implement local wait and resume mechanics without leaking thread/checkpoint semantics into public contracts?
- Can its interrupt model cleanly represent exact-action approval waits and stage sign-off waits?
- Can its interrupt model also represent `waiting_operator_input` separately from `waiting_approval` and restore more than one scoped wait at a time?
- Can it restore waits after process restart while still requiring broker validation before resuming work?
- Can it preserve `git_remote_ops` exact-action waits without collapsing them into coarse milestone waits or losing canonical request, artifact, and expected-result bindings?
- Can it preserve external audit anchor exact-action waits and deferred prepared-state resumes without losing canonical target identity, typed request hash, seal binding, or authoritative completion state?
- Can it preserve dependency-aware partial blocking so unrelated eligible work may continue when shared plan, policy, and coordination state allow it?

### Replay + Idempotency Fit
- Can LangGraph replay remain consistent with RuneCode's explicit idempotency-key model?
- Can executor side effects, checkpoint reports, result reports, and gate evidence publication remain replay-safe under LangGraph's resume semantics?
- Can replay and resume remain fail closed when git remote mutation bindings drift, including changed patch artifacts, changed expected result tree identity, or changed remote state?
- Can replay and resume remain fail closed when external audit anchor bindings drift, including changed target descriptor identity, changed typed request hash, changed target proof, or changed authoritative deferred state?
- Can replay and resume remain fail closed when validated project-substrate bindings drift for project-context-sensitive execution?

### Trust-Boundary Fit
- Does adoption preserve the rule that the runner remains untrusted and advisory only?
- Does adoption avoid creating a second local source of planning or operator truth?

### Operational Fit
- Is the dependency and persistence overhead justified for the runner package?
- Do production checkpoint backends and migrations align with RuneCode local/CI expectations?

## Expected Outcome

The likely preferred outcome is one of:

- do not adopt LangGraph because native thin-kernel orchestration remains simpler and safer
- adopt LangGraph only for internal checkpoint/wait/resume mechanics behind the runtime seam

The evaluation should explicitly avoid a third outcome where LangGraph becomes the defining runner architecture.

## Main Workstreams
- Adoption criteria and seam-fit review
- Replay/idempotency evaluation
- Approval-wait and resume-fit evaluation
- Prototype behind runtime seam if justified
- Verification against trust-boundary and broker-authority requirements
