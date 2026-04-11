## Summary
Track workflow execution as a project-level change while runner, roles, and gates ship through scoped child features that share one broker-compiled immutable `RunPlan` contract for identity, approvals, run truth, executor boundaries, and gate evidence.

## Problem
The prior combined feature mixed multiple independently deliverable components, which limited sequencing and verification granularity. It also left too much room for runner-local planning, reserved workflow/process definitions, and drift between trusted executor or gate semantics and whatever the runner would eventually infer locally.

## Proposed Change
- Keep this change as the workflow execution parent project.
- Track `CHG-2026-033-6e7b-workflow-runner-durable-state-v0` for runner and durable-state boundaries.
- Track `CHG-2026-034-b2d4-workspace-roles-v0` for role execution boundaries.
- Track `CHG-2026-035-c8e1-deterministic-gates-v0` for gate determinism and evidence.
- Freeze the long-lived foundation before filling in runtime breadth so future workflow work builds on one reviewed model:
  - broker-authoritative shared run truth with explicit runner advisory state
  - one broker-compiled immutable `RunPlan` protocol object per run
  - stable logical workflow identities plus separate retry/attempt identities
  - one public run lifecycle vocabulary with detailed coordination surfaced through run-detail models rather than a second status model
  - one approval split between exact-action approvals and stage sign-off
  - one role-kind plus executor-class matrix for workspace execution
  - one reviewed executor registry authoritative in trusted Go and projected read-only to any runner consumer
  - one typed gate contract and one typed gate-evidence model driven by explicit plan entries
  - one event-style runner-to-broker write surface for orchestration checkpoints and gate results
- Make `WorkflowDefinition` and `ProcessDefinition` operational planning inputs rather than reserved placeholders.
- Keep the runner thin: it consumes `RunPlan`, persists resumable advisory state, dispatches reviewed executors, and reports typed events.
- Keep policy, approval validity, lifecycle truth, and override legality in trusted Go.
- Explicitly avoid:
  - runner-local workflow planning
  - runner-local authorization or approval truth
  - ad hoc gate ordering or checkpoint rules
  - executor semantics forked independently in Go and TS
  - advisory runner blobs becoming a second source of operator truth

## Why Now
This work remains scheduled for v0.1.0-alpha.3 as the first honest end-to-end slice built strictly on the secure substrate, with remaining hardening and scope completed in v0.1.0-alpha.4. Freezing the `RunPlan` architecture now is the best chance to keep later workflow families, concurrency work, and new gate types from accumulating incompatible local conventions.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation details that belong in child feature changes.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Letting MVP delivery justify a weaker trust-boundary or planning foundation that later features would have to replace.

## Impact
Keeps workflow execution reviewable as a parent project with explicit feature-level ownership and sequencing while locking the shared `RunPlan`, ownership, lifecycle, executor, and gate contracts that future workflow changes must reuse.
