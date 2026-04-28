# Design

## Overview
Split the contract-first workflow-definition and binding substrate from later authoring and accelerator work.

This change now documents the recommended durable authority model for workflow execution in RuneCode:
- `ProcessDefinition` is the authoritative executable workflow graph.
- `WorkflowDefinition` is the reviewed workflow selection, packaging, and policy-binding surface over reviewed process graphs.
- The trusted control plane compiles those inputs into one immutable `RunPlan`.
- The untrusted runner executes only the compiled `RunPlan` and never reconstructs planning authority.

## Key Decisions
- `ProcessDefinition` is a typed, schema-validated executable graph surface, not a plugin system.
- `WorkflowDefinition` is not a second peer execution graph. It is the reviewed workflow-facing binding surface that selects, packages, versions, and policy-binds reviewed process graphs.
- Workflow/process definitions compose only approved existing step types and typed control-flow constructs; they do not add new privileged operations.
- Selected definitions are hash-bound through canonical bytes and bound into broker-owned signed compilation/selection evidence. `v0` does not require every checked-in definition file to become an independently signed execution authority artifact.
- JSON is the canonical on-disk and runtime format for workflow definitions, and JSON Schema is the single validation source of truth.
- Workflow/process definitions must reuse the shared workflow identity model, typed gate contract, executor model, approval split, and runner-to-broker checkpoint/result model rather than defining process-local variants.
- Workflows that compose git remote mutation must reuse shared typed git request families, signed patch artifacts, exact repository identity, and `git_remote_ops` approval semantics.
- Workflows that require dependency material must reuse the shared typed dependency-fetch and offline-cache contracts from `CHG-2026-024-acde-deps-fetch-offline-cache` rather than embedding workflow-local package-manager fetch semantics.
- Later authoring surfaces and shared-memory accelerators remain additive work on top of this contract substrate rather than being part of the contract definition itself.
- Workflow definitions that are sensitive to project context must reuse the shared project-substrate contract and validated snapshot-binding model rather than inventing workflow-local project-context references.
- Project-context-sensitive workflow selection or execution must fail closed when repository project-substrate posture is blocked.
- Workflow definitions that pause for human involvement must target the shared broker-owned wait vocabulary, including distinct `waiting_operator_input` and `waiting_approval`, rather than custom process-local wait kinds.
- Workflow/process definitions must encode enough dependency and continuation structure for broker-owned execution to block only the exact dependent scope and direct downstream work when a wait occurs.
- `v0` uses DAG-only control-flow semantics. General loops, cycles, and workflow-local re-entrant runtime semantics are out of scope for the foundational contract.
- Workflow/process definitions may encode independence and eligible continuation now, but they do not by themselves promise parallel execution in `v0`; runtime concurrency remains a broker-owned scheduling decision.
- `approval_profile` and `autonomy_posture` remain separate shared inputs to workflow selection and execution; workflow definitions must not collapse formal approval frequency and operator-guidance frequency into one mode flag.
- RuneCode keeps one topology-neutral architecture across constrained and scaled environments; performance work must optimize the same authority model rather than introducing environment-specific architecture paths.

## Authority Model

### ProcessDefinition
- `ProcessDefinition` owns executable structure.
- It should define the canonical graph of reviewed stages, steps, role scopes, typed waits, and dependency edges that the trusted control plane can compile into `RunPlan`.
- It must not become a plugin or arbitrary command surface.

### WorkflowDefinition
- `WorkflowDefinition` owns workflow-facing selection, packaging, versioning, and policy-binding over reviewed process graphs.
- It should be the object family referenced by built-in workflow packs and later custom workflow catalogs.
- It must not become a second executable graph authority that duplicates `ProcessDefinition` semantics.

### RunPlan
- `RunPlan` is the only runtime execution authority consumed by the runner.
- Broker validation, replay, evidence binding, and runner startup should eventually bind directly to the compiled `RunPlan` artifact for `{run_id, plan_id}` rather than reconstructing plan semantics by rescanning trusted workflow/process artifacts.
- If replanning is ever required, the broker must mint a new immutable `plan_id`; it must not mutate an existing compiled plan in place.

## Control-Flow Model

### Normalized Graph Shape
- The recommended foundational representation is an explicit normalized DAG with stable node identities and explicit dependency edges.
- The graph should be structured around stable logical scopes such as stages and steps rather than implicit positional nesting alone.
- The contract should favor explicit edge data over runtime inference so it can be validated, hashed, audited, diffed, and scheduled deterministically.

### DAG-Only In `v0`
- `v0` should allow no general loops or cycles.
- Retries, reruns, and recovery continue to use separate attempt identities over stable logical scopes rather than workflow-local looping semantics.
- This keeps replay, approval binding, evidence linkage, and scoped-blocking semantics fail-closed and understandable.

### Dependency-Aware Continuation
- The graph should encode which scopes depend on which upstream scopes so broker-owned orchestration can determine:
  - the exact blocked scope
  - direct downstream work that must also block
  - unrelated eligible work that may continue when policy and coordination allow
- This encoded independence is a contract-level description of what may proceed, not a promise that the runtime must execute eligible scopes concurrently in `v0`.

## Shared Contract Reuse

### Identity and Attempts
- Workflow/process definitions compile into stable logical runtime identities such as `stage_id`, `step_id`, and `role_instance_id`.
- These identities should become explicit first-class planning fields in compiled execution contracts rather than being inferred later from ad hoc gate placement alone.
- Retries and reruns use separate attempt identities rather than mutating logical scope IDs.

### Executor and Gate Reuse
- Workflow/process definitions may reference only reviewed typed executors already defined by the shared execution model.
- Workflow/process definitions must reuse the shared typed gate contract, including `gate_id`, `gate_kind`, `gate_version`, normalized inputs, and gate-evidence semantics.
- Workflow definitions must not create a second planner-owned or runner-owned executor registry.

### Control Flow and Wait Reuse
- Workflow control flow may express scoped waits and eligible continuation, but broker-owned execution state remains authoritative for whether unrelated work may proceed.
- Scoped wait semantics must compile against stable logical scope identities so blocked scope and direct downstream continuation remain auditable.
- Workflow definitions must not mint workflow-local lifecycle enums, wait vocabularies, or client-local orchestration modes.

### Approval, Audit, and Git Binding
- Workflow-defined execution must report progress through the shared runner-to-broker checkpoint/result contract.
- Stage sign-off and exact-action approval semantics remain shared and hash-bound.
- Workflow-composed git remote mutation must route through the same typed git request, patch artifact, repository identity, and exact-approval contracts as built-in git flows.
- Workflow-composed dependency fetch must route through the same typed dependency-fetch request identity, shared gateway approval semantics, broker-owned cache authority, and shared gateway audit model as built-in dependency flows.
- Approval, policy, and audit binding should use a broker-owned signed selection/compilation artifact that binds at least:
  - `workflow_definition_hash`
  - `process_definition_hash`
  - `policy_context_hash`
  - relevant execution-control inputs such as `approval_profile` and `autonomy_posture`
  - project-substrate binding when relevant

## Compilation And Persistence Model

### Canonical Definition Hashing
- `WorkflowDefinition` and `ProcessDefinition` payloads should be normalized to RFC 8785 JCS canonical JSON bytes before hashing.
- Any later authoring adapter must normalize to those same canonical bytes before validation and hashing.

### Signed Compilation/Selection Record
- The recommended first strict foundation is not independent signatures on every definition file.
- The recommended first strict foundation is a broker-owned signed compilation/selection record that binds canonical definition hashes to the exact trusted context under which execution is authorized.
- This keeps execution authority anchored in trusted compilation rather than in ambient repository presence.

### Persisted RunPlan Authority
- The compiled `RunPlan` should be persisted as a canonical broker-owned artifact and indexed by `{run_id, plan_id}`.
- Runner startup, broker-side replay checks, gate validation, and evidence linkage should consume that canonical compiled plan directly.
- Broker-side rescanning of trusted workflow/process artifacts to reconstruct runtime plan semantics is acceptable only as an interim implementation state, not as the desired durable architecture.

### Dependency-Fetch And Offline Cache Binding
- Workflow definitions may express that a stage or step requires dependency material, but they must not redefine how dependencies are fetched, cached, or materialized.
- Workflow definitions should therefore bind only to shared dependency contracts such as:
  - a lockfile-bound batch dependency request or equivalent reviewed typed dependency input
  - canonical dependency-fetch action semantics on the dedicated gateway role
  - broker-owned cache hit/miss, artifact persistence, and offline handoff behavior
- Workflow definitions must not treat raw lockfile bytes, tool-private cache directories, unpacked install trees, or package-manager-specific store paths as authoritative dependency identity.
- Workflow/process definitions should prefer reference-oriented dependency binding to reviewed typed dependency request identity or equivalent requirement references rather than embedding bulky resolver payloads into workflow definitions.
- Workflow definitions must preserve the split between:
  - gateway-backed dependency fetch and cache fill
  - ordinary workspace execution consuming broker-mediated offline cached dependencies
- Offline cached dependency use inside workflow-defined workspace execution remains ordinary workspace execution, not egress.
- The first end-to-end workflow slices should assume public-registry-first dependency fetch; workflow definitions must not require private-registry credential behavior as part of the foundational contract.

### Project-Context Binding
- Workflow definitions may reference project-context-sensitive steps or gates only through the shared project-substrate binding model established by `CHG-2026-046-a91d-runecontext-verified-project-substrate-compatibility-lifecycle-v0`.
- When project context matters, selected workflow execution should carry the validated project-substrate snapshot identity used for policy, audit, and later verification binding.
- The recommended location for that execution binding is the compiled `RunPlan`, which should carry the exact validated project-substrate digest used for execution-sensitive context.
- Resume and continuation of project-context-sensitive execution should fail closed on incompatible project-substrate drift rather than relying on ambient repository state or summary fields alone.
- Workflow definitions must not embed alternate project discovery, init, adopt, or upgrade semantics.

## Performance And Topology-Neutrality

- RuneCode should keep one overall architecture across constrained local hardware and larger deployments.
- Performance must come from efficient implementation of that shared architecture rather than from separate small-device and scaled-deployment contract models.
- The foundation should therefore optimize for:
  - compact canonical definitions and compiled plans
  - explicit IDs and adjacency data instead of repeated runtime inference
  - direct use of persisted compiled plan artifacts instead of repeated artifact rescans and re-unmarshal paths
  - bounded and configurable execution concurrency under broker control
  - streaming IO and CAS-oriented persistence for large artifacts
  - topology-neutral identity that never depends on host-local paths, runner-local cache layout, or deployment shape
- The contract should describe eligible continuation and authority boundaries in the same way everywhere, whether actual execution later runs on a Raspberry Pi or a horizontally scaled deployment.

## Schema Evolution Guidance

- This feature should treat the workflow-definition substrate as a real contract evolution, not as an informal extension of placeholder schemas.
- Where `WorkflowDefinition`, `ProcessDefinition`, or `RunPlan` must gain richer graph, scope, or binding semantics, the schema-version changes should be explicit and intentional.
- Read models should expose plan- and process-relevant identity when those fields become operationally significant for replay safety, approval validity, or operator diagnostics.

## Main Workstreams
- Workflow vs Process Authority Split.
- `ProcessDefinition` Graph Contract.
- `WorkflowDefinition` Selection/Binding Contract.
- Validation + Canonicalization.
- Persisted `RunPlan` Compilation Authority.
- Policy, Approval, Audit, Dependency, and Git Binding.
- Shared Project-Context Binding.
- Contract Split from Later Authoring + Accelerators.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
