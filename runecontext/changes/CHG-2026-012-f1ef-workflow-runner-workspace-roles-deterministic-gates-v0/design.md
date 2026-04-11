# Design

## Overview
Use this change as the project-level tracker for secure workflow execution while implementation lands in child feature changes.

The parent project owns the shared integration contracts that must stay consistent across runner orchestration, workspace execution, and gate enforcement. The goal is to freeze these seams early enough that later workflow extensibility, shared-workspace concurrency, and new workflow families extend one model instead of inventing parallel status, approval, executor, or evidence vocabularies.

The recommended foundation is a broker-compiled plan architecture:
- trusted Go compiles, authorizes, validates, and records
- the untrusted runner executes one immutable `RunPlan` and reports typed events
- broker-owned state remains canonical
- runner-owned state remains resumable and advisory only

## Key Decisions
- Child features own runtime implementation detail.
- Parent project owns sequencing and integration posture.
- Security invariants remain untrusted-runner, policy enforcement, and evidence-backed execution.
- Integration posture includes one shared broker logical API vocabulary for runs, approvals, audit posture, and operator-visible blocked state across child features.
- Integration posture also includes one shared policy vocabulary for canonical action identity, role taxonomy, gateway destination semantics, and exact-action-vs-stage-sign-off approval behavior across child features.
- Integration posture also includes one shared runtime-isolation contract vocabulary across child features:
  - backend-neutral launch/session/attachment seams
  - runtime posture vocabulary (`backend_kind`, runtime isolation assurance, provisioning/binding posture, audit posture)
  - authoritative launcher/broker runtime-state projection
- Broker remains the authoritative owner of shared run truth, approval truth, operator-facing lifecycle state, and gate/evidence linkage.
- Broker compiles one immutable `RunPlan` per run from workflow/process definitions, policy inputs, reviewed executor bindings, and planned gates. If future replanning is ever required, it must mint a new superseding plan identity rather than mutating an existing plan in place.
- `WorkflowDefinition` and `ProcessDefinition` should become operational planning inputs rather than reserved placeholders.
- Runner durable state remains untrusted and resumable, but it is limited to orchestration journals, checkpoints, local replay metadata, and plan-bound scheduler state; runner state may be exposed only through explicit advisory broker surfaces.
- Stable logical identities are shared across child features:
  - `run_id` names one run instance
  - `stage_id`, `step_id`, and `role_instance_id` name stable logical workflow scopes
  - retry and recovery mint separate attempt identities instead of redefining logical scope identities
- Public lifecycle stays on one shared broker vocabulary (`pending`, `starting`, `active`, `blocked`, `recovering`, `completed`, `failed`, `cancelled`). Partial blocking or branch-local waits must surface through `RunDetail`, `RunStageSummary`, `RunRoleSummary`, and `RunCoordinationSummary` instead of a second public lifecycle enum.
- Approval creation remains policy-derived and broker-materialized. Exact-action approvals bind one canonical `ActionRequest` hash; stage sign-off binds one canonical stage summary hash and must be superseded when that summary changes.
- Workspace role kind and executor class remain distinct dimensions. Role kind describes least-privilege function; executor class describes action risk. Child features must not collapse them into one overloaded role label.
- Non-shell-passthrough execution means reviewed typed executors with explicit contract shapes, not freeform command strings or raw shell interpreters hidden behind wrappers.
- One reviewed executor registry should remain authoritative in the trusted control plane. Any runner-visible copy is a read-only projection for dispatch validation and must not become a second policy authority.
- Gates are first-class typed workflow checks with stable identity, deterministic inputs, explicit retry/override semantics, and typed evidence objects rather than ad hoc log scraping.
- Runner-to-broker orchestration writes should be event-style typed checkpoint/result reports that the broker validates and projects, not blind runner-owned state upserts.
- Runner durable-state persistence should use a versioned append-first journal plus snapshots with deterministic broker-wins reconciliation after restart.
- The runner should stay thin: plan loader, broker client, journal/snapshot store, scheduler, executor adapters, and report emitter. It must not become a planner, policy engine, or second lifecycle source.

## Shared Integration Contracts

### RunPlan Contract
- Child features should share one immutable `RunPlan` protocol object compiled by the broker.
- `RunPlan` should carry at least:
  - stable logical scope identities for stages, steps, roles, and gates
  - reviewed executor bindings and executor-class expectations
  - explicit gate placements, order, and retry posture
  - approval checkpoints and scope bindings
  - hashes or digest refs for the workflow, process, manifest, and policy inputs used to compile the plan
- Runner journal records, checkpoint reports, result reports, gate evidence, and future audit linkage should bind to the plan identity so recovery and later feature work do not guess which plan shaped a run.

### Run Truth Ownership
- Broker-owned state is authoritative for:
  - shared run lifecycle and coordination posture
  - persisted policy decisions and approvals
  - artifact and gate-evidence linkage
  - operator-facing read models
- Launcher-owned evidence projected by broker is authoritative for backend/runtime posture.
- Runner-owned state is advisory only and should contain resumable orchestration mechanics rather than a second source of operator-visible truth.

### Identity Model
- Child features should reuse one identity model across policy, broker, runner, launcher, and audit surfaces.
- Stable workflow scope identities should survive retries and restarts.
- Retries, gate reruns, and step reruns should mint new attempt identities instead of mutating prior logical scope identity.

### Approval Model
- Exact-action approval is the narrowest approval form and should be used when one immutable action request must be reviewed or consumed once.
- Stage sign-off is the stage-bound checkpoint form and should authorize continuation only while the bound stage summary hash remains current.
- Approval delivery channels are advisory only; approval identity, request hash, decision hash, and policy bindings remain canonical across child features.

### Execution Boundary Model
- Workspace roles remain offline and least-privilege.
- Executor registry and policy matrix should be shared inputs, not duplicated inside each workflow or gate implementation.
- System-modifying execution must not leak into ordinary workspace execution through wrappers, shell passthrough, or feature-local exceptions.
- Plan-bound executor dispatch should remain the only ordinary execution path for the runner.

### Gate Model
- Gates should be addressed by stable gate identity and version rather than only by human-facing names.
- Gate inputs should be declared and normalized deterministically.
- Gate ordering and checkpoint placement should come from workflow/process planning and be compiled into `RunPlan`, not rediscovered by runner-local convention.
- Gate evidence should be reference-heavy, content-addressed, and reusable by audit, broker read models, and future retention policies.

## Avoided Foundation Shortcuts
- Do not let the runner invent workflow order or choose unplanned work opportunistically.
- Do not fork authorization semantics between trusted Go and the runner.
- Do not leave `WorkflowDefinition` and `ProcessDefinition` as reserved-only shells while runtime behavior grows elsewhere.
- Do not rely on shell wrappers or command heuristics as the primary execution-authority model.
- Do not let free-form advisory state become the long-term operator contract.

## Main Workstreams
- `CHG-2026-033-6e7b-workflow-runner-durable-state-v0`
- `CHG-2026-034-b2d4-workspace-roles-v0`
- `CHG-2026-035-c8e1-deterministic-gates-v0`
- Cross-feature `RunPlan` contract and operational workflow/process planning inputs
- Integration milestone: minimal end-to-end demo run across child features using a broker-compiled immutable `RunPlan`

## Cross-Feature Outcomes To Preserve
- One immutable broker-compiled `RunPlan` contract reused by later workflow families.
- One typed runner->broker checkpoint/result contract reused by later workflow families.
- One stable logical workflow identity model reused by policy, approvals, artifacts, and future shared-workspace coordination.
- One public lifecycle model with richer detail surfaces instead of layered UI-only or runner-only status taxonomies.
- One approval split between exact-action approval and stage sign-off, with broker materialization from policy decisions.
- One reviewed workspace executor registry and role-to-executor policy matrix.
- One typed gate-evidence contract reusable by audit and future retention/override work.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
