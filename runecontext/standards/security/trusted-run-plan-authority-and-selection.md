---
schema_version: 1
id: security/trusted-run-plan-authority-and-selection
title: Trusted Run Plan Authority And Selection
status: active
suggested_context_bundles:
    - go-control-plane
    - runner-boundary
    - protocol-foundation
---

# Trusted Run Plan Authority And Selection

When RuneCode compiles workflow execution contracts:

- `WorkflowDefinition` is the reviewed workflow-facing selection and policy-binding surface
- `ProcessDefinition` is the authoritative executable DAG surface
- `RunPlan` is the only runner-consumed execution authority
- The runner may validate `RunPlan` shape for self-protection, but it must not reconstruct planning truth from workflow or process inputs

Trusted compilation and persistence requirements:

- Compile `RunPlan` in the trusted control plane from reviewed workflow and process artifacts plus trusted policy and project-context inputs
- Persist the compiled `RunPlan` artifact durably before treating it as active execution authority
- Persist authority records that bind at least `run_id`, `plan_id`, `run_plan_digest`, workflow/process definition hashes, policy context hash, validated project-context digest when required, and the compiled gate-entry set used for replay and validation
- Persist compilation records that bind the trusted source artifact references plus durable binding and record digests for later audit and verification
- Treat `RunPlan` entries, dependency edges, gate identity, and executor bindings as compiled authority owned by the trusted control plane rather than as runner-derived convenience data

Active-plan selection requirements:

- Select the active trusted run plan per run from persisted authority records, not from ambient repository state or runner-local hints
- Use `supersedes_plan_id` to make immutable replanning explicit; replanning must mint a new `plan_id` instead of mutating an existing plan in place
- Fail closed when active-plan selection is ambiguous, when conflicting records exist for the same `{run_id, plan_id}`, or when no active trusted run plan can be resolved
- Treat a superseded plan as inactive for replay, validation, scheduling, and evidence binding

Project-context and drift requirements:

- Bind project-context-sensitive execution to the validated `project_context_identity_digest` carried by the active trusted run plan
- Fail closed when a trusted run plan requires validated project context and no current validated digest is available
- Fail closed when the current validated project-context digest drifts from the digest bound into the active trusted run plan
- Do not recover project-context-sensitive execution by rescanning repository files or accepting runner-local summaries as equivalent authority

Replay and validation requirements:

- Reconcile restart, replay, gate validation, and gate-evidence binding against the persisted active trusted run plan rather than recompiling from ambient definitions during routine execution
- Bind gate validation and evidence linkage to the active plan's gate entry identity, retry posture, expected input digests, dependency-cache handoff requirements, and workflow/process/policy/project-context digests
- Treat cached plan projections as disposable performance aids only; the persisted authority record remains canonical

Treat a change as risky if it lets runner-local state, rescanned repository contents, or mutable in-place planning override persisted trusted run-plan authority.
