# Tasks

## Workflow vs Process Authority Split

- [ ] Make `ProcessDefinition` the authoritative executable workflow graph surface.
- [ ] Make `WorkflowDefinition` the reviewed workflow selection, packaging, and policy-binding surface over reviewed process graphs.
- [ ] Remove or prevent any interpretation where `WorkflowDefinition` and `ProcessDefinition` remain two near-peer executable authorities.
- [ ] Record the split explicitly so later authoring, workflow packs, and custom workflow catalogs build on it instead of reopening it.

## `ProcessDefinition` Graph Contract

- [ ] Define the `ProcessDefinition` object family as a typed executable graph for workflow composition.
- [ ] Limit it to approved existing step types and typed control-flow constructs with no new privileged operations.
- [ ] Represent executable structure as a normalized DAG with stable scope identities and explicit dependency edges.
- [ ] Keep `v0` DAG-only; forbid general loops, cycles, and workflow-local re-entrant control-flow semantics.
- [ ] Reuse the shared workflow identity model so workflow definitions compile into stable logical `stage_id`, `step_id`, and `role_instance_id` values.
- [ ] Make stable logical scope IDs explicit first-class planning fields rather than leaving them implicit in gate ordering alone.
- [ ] Keep retries and reruns on separate attempt identities rather than mutating logical scope IDs.

## `WorkflowDefinition` Selection/Binding Contract

- [ ] Define the `WorkflowDefinition` object family as the reviewed workflow-facing binding surface over one or more reviewed process graphs.
- [ ] Keep workflow-facing packaging/versioning/selection semantics distinct from executable graph semantics.
- [ ] Keep `approval_profile` and `autonomy_posture` as separate shared inputs rather than collapsing them into one workflow-local mode.

## Validation + Canonicalization

- [ ] Keep JSON as the canonical on-disk and runtime format.
- [ ] Use JSON Schema as the single validation source of truth.
- [ ] Normalize any future authoring adapters to the same RFC 8785 JCS canonical JSON bytes before validation and hashing.
- [ ] Treat the richer workflow/process substrate as an explicit schema evolution with intentional schema-version updates where contract shape changes materially.

## Compilation And Runtime Authority

- [ ] Keep selected workflow/process definitions hash-bound through canonical JCS bytes.
- [ ] Bind selected definitions into a broker-owned signed compilation/selection record rather than requiring every definition file to be an independently signed execution authority artifact in `v0`.
- [ ] Make `RunPlan` the only runtime execution authority consumed by the runner.
- [ ] Persist the compiled `RunPlan` as a canonical broker-owned artifact indexed by `{run_id, plan_id}`.
- [ ] Move broker-side validation, replay, and evidence binding toward direct compiled-plan consumption rather than trusted-artifact rescans that reconstruct plan semantics on demand.

## Shared Contract Binding

- [ ] Restrict workflow/process definitions to reviewed typed executors already defined by the shared execution model.
- [ ] Reuse the shared typed gate contract, including gate identity/version, gate-attempt semantics, and gate-evidence linkage.
- [ ] Bind selected workflow/process definitions into policy evaluation, approval requests, and audit evidence.
- [ ] Route workflow execution progress through the shared runner-to-broker checkpoint/result model rather than a workflow-local status channel.
- [ ] Reuse the shared human-involvement wait vocabulary, including distinct `waiting_operator_input` and `waiting_approval`, rather than defining workflow-local wait kinds.
- [ ] Encode dependency-aware continuation so shared execution can distinguish blocked scope and direct downstream work from unrelated eligible work.
- [ ] Keep encoded independence as a contract-level eligibility model, not as a promise of parallel execution in `v0`.
- [ ] Ensure workflow-composed git remote mutation still routes through shared typed git request families, signed patch artifacts, exact repository identity, and `git_remote_ops` exact-action approval.
- [ ] Prefer reviewed git-operation intent and trusted compilation into shared git request families over embedding workflow-local raw git mutation payloads.
- [ ] Ensure workflow-composed dependency fetch routes through the shared typed dependency-fetch request/batch contracts, broker-owned cache authority, and shared gateway audit/approval semantics from `CHG-2026-024-acde-deps-fetch-offline-cache`.
- [ ] Prefer reference-oriented dependency requirement binding to shared typed dependency request identity over embedding raw lockfile bytes or bulky workflow-local resolver payloads.
- [ ] Keep workflow definitions from embedding raw lockfile bytes, tool-private cache paths, unpacked dependency trees, or package-manager-specific materialization paths as authoritative dependency identity.
- [ ] Keep offline cached dependency use modeled as broker-mediated internal artifact handoff plus ordinary workspace execution rather than as workflow-local egress behavior.
- [ ] Keep the initial contract compatible with the public-registry-first first slice and avoid dependency semantics that require private-registry credential flows in the foundational workflow substrate.
- [ ] Reuse the shared project-substrate contract and validated snapshot-binding model for project-context-sensitive workflow selection or execution.
- [ ] Fail closed for project-context-sensitive workflow execution when repository project-substrate posture is missing, invalid, non-verified, or unsupported.
- [ ] Bind project-context-sensitive compiled plans to the exact validated project-substrate snapshot digest used for execution-sensitive context.
- [ ] Fail closed on incompatible project-substrate drift during resume or continuation of project-context-sensitive execution.
- [ ] Forbid workflow definitions from embedding alternate project discovery, init, adopt, or upgrade semantics.

## Performance And Topology-Neutrality

- [ ] Preserve one topology-neutral authority model across constrained local devices and larger deployments; do not introduce separate architecture paths for small-device versus scaled environments.
- [ ] Keep plan and definition contracts compact, explicit, and indexable so broker and runner hot paths avoid repeated inference work.
- [ ] Reduce or eliminate repeated full trusted-artifact rescans, `ReadAll`, unmarshal, and sort/rebuild work in runtime validation paths once canonical compiled-plan persistence exists.
- [ ] Keep concurrency bounded and broker-owned rather than making it an implicit property of workflow definition shape.
- [ ] Keep object identity independent from host-local paths, runner-local cache layout, and deployment topology.

## Split Discipline

- [ ] Keep generic workflow authoring UX out of this contract-first change.
- [ ] Keep shared-memory accelerators out of this contract-first change.
- [ ] Keep implementation-track decomposition and git-worktree execution as later additive work on top of this substrate rather than a second workflow planning format.

## Acceptance Criteria

- [ ] Workflow definitions remain schema-validated, hash-bound, and auditable.
- [ ] `ProcessDefinition` is the authoritative executable graph, `WorkflowDefinition` is the reviewed selection/binding surface, and `RunPlan` is the only runtime execution authority.
- [ ] Workflow definitions do not add new privileged operations or weaken trust boundaries.
- [ ] `v0` workflow/process control flow is explicit DAG-based structure with no general loop semantics.
- [ ] Built-in and later custom workflows can share one definition and binding substrate.
- [ ] Later authoring and accelerator work can extend this substrate without redefining its authority model.
- [ ] Later track decomposition and isolated worktree execution can extend this substrate without introducing a second planning model.
- [ ] Project-context-sensitive workflows share one validated project-substrate binding model instead of inventing workflow-local project-context semantics.
- [ ] Workflow-defined waits preserve shared scoped blocking semantics instead of forcing a whole-workflow stop model.
- [ ] Encoded dependency-aware independence can support later continuation and scheduling work without promising new parallel-execution semantics in `v0`.
- [ ] Workflow-defined dependency needs reuse the shared dependency-fetch and offline-cache substrate rather than inventing workflow-local package-manager or cache authority semantics.
- [ ] The same control-plane architecture remains valid across constrained local hardware and larger deployments, with performance tuning implemented without changing trust or contract semantics.
