---
schema_version: 1
id: security/runner-durable-state-and-replay
title: Runner Durable State And Replay
status: active
suggested_context_bundles:
    - runner-boundary
    - protocol-foundation
---

# Runner Durable State And Replay

When the untrusted runner persists resumable orchestration state for plan execution:

- Treat runner durable state as advisory orchestration state only; broker-owned run truth, approval truth, gate outcomes, and operator-facing lifecycle remain canonical in trusted services
- Bind every durable journal, snapshot, wait record, and replay decision to the active `{run_id, plan_id}` pair; fail closed if persisted state does not match the broker-compiled immutable plan identity
- Treat the active `{run_id, plan_id}` pair as the persisted trusted selection result, not just a runner-local label; fail closed when the bound plan has been superseded or when trusted plan selection becomes ambiguous
- Use append-first journal families plus snapshots rather than one mutable status blob; snapshots are a recovery cache, not a second source of truth
- Put explicit schema versions on journal and snapshot families and fail closed on unknown future versions instead of guessing compatibility
- Keep stable logical scope identity separate from retry and replay identity:
  - stable scope: `stage_id`, `step_id`, `role_instance_id`
  - attempt/replay identity: `stage_attempt_id`, `step_attempt_id`, `gate_attempt_id`, stable idempotency keys
- Require replay-safe idempotency for every externally visible side effect, including runner checkpoint/result reports, approval-wait transitions, gate evidence linkage, and executor-dispatch side effects that can survive process failure
- On restart or resume, reconcile persisted runner state against trusted broker state explicitly; broker-canonical run, plan, and approval bindings win over runner-local hints
- On restart or resume, also reconcile the current validated project-context digest against the active trusted run plan when execution is project-context-sensitive; drift fails closed rather than silently resuming under ambient repository state
- Approval waits must persist enough trusted correlation data to resume safely:
  - canonical `approval_id`
  - `run_id` and `plan_id`
  - binding kind
  - bound action hash or bound stage summary hash
  - blocked scope identity
  - stable wait-entered and wait-cleared idempotency keys
- Do not resume blocked work solely because runner-local persistence says it was parked; resume only after broker validation confirms the same approval remains valid for the same bound scope/hash and active plan identity
- Approval waits may block only the exact bound scope; unrelated eligible work may continue only when the active plan and trusted coordination state still allow it
- Keep runner-internal lifecycle detail and partial-blocking mechanics internal or advisory; do not mint a second public lifecycle vocabulary outside broker-owned read models
- Any internal runtime seam may cover only local checkpoint, wait, restore, and resume mechanics; it must not let a framework or third-party runtime become a second planning authority, approval authority, lifecycle authority, or public contract source

Treat a change as risky if it weakens fail-closed replay, plan-bound identity, broker-wins reconciliation, or restart-safe approval validation.
