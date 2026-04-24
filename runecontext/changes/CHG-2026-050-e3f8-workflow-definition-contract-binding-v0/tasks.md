# Tasks

## `ProcessDefinition` Contract

- [ ] Define the `ProcessDefinition` object family for workflow composition.
- [ ] Limit it to approved existing step types and typed control-flow constructs with no new privileged operations.
- [ ] Keep selected workflow definitions as signed, hash-bound inputs to policy, approval, and audit flows.
- [ ] Reuse the shared workflow identity model so workflow definitions compile into stable logical `stage_id`, `step_id`, and `role_instance_id` values.
- [ ] Keep retries and reruns on separate attempt identities rather than mutating logical scope IDs.

## Validation + Canonicalization

- [ ] Keep JSON as the canonical on-disk and runtime format.
- [ ] Use JSON Schema as the single validation source of truth.
- [ ] Normalize any future authoring adapters to the same RFC 8785 JCS canonical JSON bytes before validation and hashing.

## Shared Contract Binding

- [ ] Restrict workflow definitions to reviewed typed executors already defined by the shared execution model.
- [ ] Reuse the shared typed gate contract, including gate identity/version, gate-attempt semantics, and gate-evidence linkage.
- [ ] Bind selected workflow definitions into policy evaluation, approval requests, and audit evidence.
- [ ] Route workflow execution progress through the shared runner-to-broker checkpoint/result model rather than a workflow-local status channel.
- [ ] Ensure workflow-composed git remote mutation still routes through shared typed git request families, signed patch artifacts, exact repository identity, and `git_remote_ops` exact-action approval.
- [ ] Reuse the shared project-substrate contract and validated snapshot-binding model for project-context-sensitive workflow selection or execution.
- [ ] Reuse the shared human-involvement wait vocabulary, including distinct `waiting_operator_input` and `waiting_approval`, rather than defining workflow-local wait kinds.
- [ ] Encode dependency-aware continuation so shared execution can distinguish blocked scope and direct downstream work from unrelated eligible work.
- [ ] Keep `approval_profile` and `autonomy_posture` as separate shared inputs rather than collapsing them into one workflow-local mode.
- [ ] Fail closed for project-context-sensitive workflow execution when repository project-substrate posture is missing, invalid, non-verified, or unsupported.
- [ ] Forbid workflow definitions from embedding alternate project discovery, init, adopt, or upgrade semantics.

## Split Discipline

- [ ] Keep generic workflow authoring UX out of this contract-first change.
- [ ] Keep shared-memory accelerators out of this contract-first change.
- [ ] Record the split explicitly so later authoring and accelerator work builds on this substrate rather than reopening it.

## Acceptance Criteria

- [ ] Workflow definitions remain schema-validated, hash-bound, and auditable.
- [ ] Workflow definitions do not add new privileged operations or weaken trust boundaries.
- [ ] Built-in and later custom workflows can share one definition and binding substrate.
- [ ] Later authoring and accelerator work can extend this substrate without redefining its authority model.
- [ ] Project-context-sensitive workflows share one validated project-substrate binding model instead of inventing workflow-local project-context semantics.
- [ ] Workflow-defined waits preserve shared scoped blocking semantics instead of forcing a whole-workflow stop model.
