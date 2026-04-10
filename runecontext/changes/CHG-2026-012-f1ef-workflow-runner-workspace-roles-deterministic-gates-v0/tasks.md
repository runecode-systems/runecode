# Tasks

## Child Feature Tracking

- [ ] Track `CHG-2026-033-6e7b-workflow-runner-durable-state-v0` to completion.
- [ ] Track `CHG-2026-034-b2d4-workspace-roles-v0` to completion.
- [ ] Track `CHG-2026-035-c8e1-deterministic-gates-v0` to completion.

## Cross-Feature Coordination

- [ ] Keep child feature sequencing aligned with `CHG-2026-007-2315-policy-engine-v0` and `CHG-2026-008-62e1-broker-local-api-v0`.
- [ ] Keep child feature sequencing aligned with `CHG-2026-009-1672-launcher-microvm-backend-v0`.
- [ ] Keep the minimal end-to-end demo run as the integration milestone across child features.
- [ ] Keep run lifecycle, approval lifecycle, blocked-state semantics, and operator-facing read models aligned across broker, runner, and TUI work.
- [ ] Keep canonical action identity, role taxonomy, gateway destination semantics, and exact-action-vs-stage-sign-off approval behavior aligned across child features.
- [ ] Keep backend-neutral launch/session/attachment contracts, runtime posture vocabulary, and authoritative launcher/broker runtime-state projection aligned across child features.
- [ ] Freeze one shared run-truth ownership model across child features:
  - broker owns canonical shared run truth and operator-facing projections
  - launcher supplies authoritative runtime evidence projected by broker
  - runner owns only resumable orchestration detail and advisory checkpoint metadata
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
- [ ] Parent-project docs capture the shared ownership, identity, lifecycle, approval, executor, gate, and reconciliation contracts that future workflow changes must reuse.
