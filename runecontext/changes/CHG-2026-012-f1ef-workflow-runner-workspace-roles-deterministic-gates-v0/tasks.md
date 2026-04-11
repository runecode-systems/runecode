# Tasks

## Child Feature Tracking

- [ ] Track `CHG-2026-033-6e7b-workflow-runner-durable-state-v0` to completion.
- [ ] Track `CHG-2026-034-b2d4-workspace-roles-v0` to completion.
- [ ] Track `CHG-2026-035-c8e1-deterministic-gates-v0` to completion.

## Foundation Contracts

- [ ] Freeze one immutable broker-compiled `RunPlan` protocol object shared by runner, workspace-role execution, and deterministic gates.
- [ ] Turn `WorkflowDefinition` and `ProcessDefinition` into operational planning inputs rather than reserved-only placeholders.
- [ ] Freeze one shared rule that policy, approval validity, lifecycle truth, and override legality remain trusted-domain responsibilities.
- [ ] Freeze one shared rule that the runner consumes `RunPlan` and reports typed events, but does not plan workflows locally.
- [ ] Freeze one reviewed executor registry authoritative in trusted Go, with any runner-visible copy treated as read-only projection only.
- [ ] Freeze one shared rule that future replanning mints a new superseding plan identity instead of mutating an existing `RunPlan` in place.

## Cross-Feature Coordination

- [ ] Keep child feature sequencing aligned with `CHG-2026-007-2315-policy-engine-v0` and `CHG-2026-008-62e1-broker-local-api-v0`.
- [ ] Keep child feature sequencing aligned with `CHG-2026-009-1672-launcher-microvm-backend-v0`.
- [ ] Keep the minimal end-to-end demo run as the integration milestone across child features, and require it to execute from a broker-compiled immutable `RunPlan`.
- [ ] Keep run lifecycle, approval lifecycle, blocked-state semantics, and operator-facing read models aligned across broker, runner, and TUI work.
- [ ] Keep canonical action identity, role taxonomy, gateway destination semantics, and exact-action-vs-stage-sign-off approval behavior aligned across child features.
- [ ] Keep backend-neutral launch/session/attachment contracts, runtime posture vocabulary, and authoritative launcher/broker runtime-state projection aligned across child features.
- [ ] Freeze one shared run-truth ownership model across child features:
  - broker owns canonical shared run truth and operator-facing projections
  - launcher supplies authoritative runtime evidence projected by broker
  - runner owns only resumable orchestration detail and advisory checkpoint metadata
- [ ] Keep runner architecture thin across child features:
  - plan loader
  - broker client
  - journal/snapshot store
  - scheduler
  - executor adapters
  - report emitter
- [ ] Explicitly avoid foundation shortcuts across child features:
  - runner-local workflow planning
  - runner-local authorization or approval truth
  - ad hoc gate ordering rules
  - executor semantics forked separately in Go and TS
  - advisory blobs becoming a second operator contract
- [ ] Freeze one shared workflow identity model across child features:
  - stable logical `run_id`, `stage_id`, `step_id`, and `role_instance_id`
  - separate attempt identities for retries, gate reruns, and recovery
- [ ] Freeze one shared lifecycle mapping rule so public run state stays on the broker lifecycle vocabulary and richer partial-blocking detail lives in run-detail coordination surfaces.
- [ ] Freeze one shared approval model so broker-materialized approvals preserve the policy split between exact-action approvals and stage sign-off across runner, gate, and TUI work.
- [ ] Freeze one shared execution-boundary model so workspace `role_kind` and `executor_class` remain distinct and reviewed together.
- [ ] Freeze one shared gate contract and gate-evidence model so later workflow families and retention logic reuse the same typed evidence surface.
- [ ] Freeze one shared event-style runner->broker write contract for orchestration checkpoints, approval waits, gate results, and lifecycle reports.
- [ ] Keep runner durable-state versioning, replay, and reconciliation aligned with broker-owned run truth so recovery behavior remains deterministic across future workflow-extensibility and concurrency work.

## Acceptance Criteria

- [ ] Child features remain linked, sequenced, and aligned to the same workflow security invariants.
- [ ] Parent-project status remains an accurate integration view rather than duplicating feature implementation detail.
- [ ] Parent-project docs capture the shared ownership, `RunPlan`, identity, lifecycle, approval, executor, gate, and reconciliation contracts that future workflow changes must reuse.
- [ ] The project foundation is strong enough for future workflow features without requiring a later rewrite away from runner-local planning or ad hoc gate/executor conventions.
