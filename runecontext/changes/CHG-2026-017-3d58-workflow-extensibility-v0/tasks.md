# Tasks

## `ProcessDefinition` Contract

- [ ] Define the `ProcessDefinition` object family for custom workflow composition.
- [ ] Limit it to approved existing step types and typed control-flow constructs; no new privileged operations.
- [ ] Keep selected process definitions as signed, hash-bound inputs to policy, approval, and audit flows.
- [ ] Reuse the shared workflow identity model so process definitions compile into stable logical `stage_id`, `step_id`, and `role_instance_id` values rather than process-local ad hoc identifiers.
- [ ] Keep retries and reruns on separate attempt identities rather than mutating logical scope IDs.
- [ ] Restrict custom workflows to reviewed typed executors already defined by the shared execution model; no process-local shell passthrough or unreviewed executor contracts.
- [ ] Reuse the shared typed gate contract, including gate identity/version, gate-attempt semantics, and gate-evidence linkage.
- [ ] If custom workflows compose git remote mutation, restrict them to the reviewed typed git request families and signed patch artifact contracts rather than process-local git payloads or ad hoc remote operations.
- [ ] Forbid process definitions from directly mutating repository policy truth, ref allowlists, or repository-specific commit policy through local settings or untyped side channels.

## Validation + Canonicalization

- [ ] Keep JSON as the canonical on-disk and runtime format.
- [ ] Use JSON Schema as the single validation source of truth.
- [ ] Normalize any future authoring adapters to the same RFC 8785 JCS canonical JSON bytes before validation and hashing, keeping authored workflow definitions object-rooted.

## Shared-Memory Accelerators

- [ ] Define rebuildable shared-memory accelerators for derived artifacts only.
- [ ] Keep authoritative state in the run DB, artifact store, and audit trail.

## Policy, Approval, and Audit Binding

- [ ] Bind selected process definitions into policy evaluation, approval requests, and audit evidence.
- [ ] Ensure custom workflows cannot bypass manifest, broker, or policy enforcement.
- [ ] Preserve the shared approval split between exact-action approvals and stage sign-off for custom workflows.
- [ ] Ensure stage-summary changes in custom workflows supersede stale sign-off requests using the same shared hash-bound semantics as built-in workflows.
- [ ] Route process execution progress through the shared runner->broker checkpoint/result model rather than a process-local status/update channel.
- [ ] Ensure workflow-composed git remote mutation still routes through `git_remote_ops` exact-action approval rather than workflow-local milestone or stage approval.
- [ ] Ensure workflow-composed git remote mutation binds canonical repository identity, target refs, referenced patch artifact digests, expected result tree hash, and canonical action request hash.

## Authoring + UX Surfaces

- [ ] Define authoring and review surfaces for process definitions.
- [ ] Keep machine validation deterministic and explicit.

## Acceptance Criteria

- [ ] Custom workflows remain schema-validated, hash-bound, and auditable.
- [ ] Workflow customization does not add new privileged operations or weaken existing trust boundaries.
- [ ] Custom workflows reuse the shared identity, executor, gate, approval, git request, patch artifact, and runner->broker execution contracts rather than inventing parallel workflow semantics.
