# Tasks

## Authoring and Review Surfaces

- [ ] Define generic authoring and review surfaces for workflow definitions on top of the contract-first substrate.
- [ ] Define generic authoring and review surfaces for both `WorkflowDefinition` selection/packaging content and `ProcessDefinition` executable graph content.
- [ ] Keep authoring flows deterministic and explicit rather than plugin-like or freeform.
- [ ] Ensure authoring and review surfaces do not become a second source of authoritative workflow truth.
- [ ] Define an explicit separate registration/catalog path for custom workflows rather than repository-local override or shadowing of product-shipped built-in workflow IDs.
- [ ] Ensure authoring surfaces express execution semantics only through canonical fields compatible with `CHG-2026-048-6b7a-session-execution-orchestration-v0`, including shared wait vocabulary and separate `approval_profile` versus `autonomy_posture` inputs.
- [ ] Ensure authoring preserves the CHG-050 `v0` DAG-only executable-graph posture and does not introduce loops, cycles, or authoring-local re-entrant semantics.
- [ ] Where authoring targets draft-like workflows, preserve the CHG-049 artifact-first plus explicit promote/apply posture rather than collapsing draft generation and canonical mutation into one ambient edit side effect.

## Authoring Adapter Normalization

- [ ] Normalize any authoring adapters to the same canonical workflow-definition/process-definition formats and hashing rules established by `CHG-2026-050-e3f8-workflow-definition-contract-binding-v0`.
- [ ] Keep machine validation deterministic and explicit.

## Shared-Memory Accelerators

- [ ] Define rebuildable shared-memory accelerators for derived artifacts only.
- [ ] Keep authoritative state in the run DB, artifact store, and audit trail.

## Safe Adoption UX

- [ ] Ensure generic workflow authoring continues to reuse the shared identity, executor, gate, approval, audit, and git-composition contracts from `CHG-2026-050-e3f8-workflow-definition-contract-binding-v0`.
- [ ] Ensure generic workflow authoring does not bypass broker-owned signed compilation/selection binding or immutable `RunPlan` runtime authority.
- [ ] Ensure custom workflow authoring cannot mutate repository policy truth, ref allowlists, or repository-specific commit policy through local settings or untyped side channels.
- [ ] Ensure custom workflow authoring reuses the shared validated project-substrate snapshot-binding model where project context is relevant.
- [ ] Ensure custom workflow authoring cannot invent alternate project discovery, init, adopt, or upgrade semantics through local settings or untyped side channels.
- [ ] Ensure custom workflow authoring cannot override or shadow product-shipped built-in workflow identities from CHG-049.
- [ ] Ensure any explicit implementation-track authoring targets canonical track declarations that `CHG-2026-051-4b9d-implementation-track-decomposition-git-worktree-execution-v0` can consume without making authoring UX the scheduler authority.

## Acceptance Criteria

- [ ] Generic workflow authoring and review remain deterministic and build on the contract-first workflow substrate rather than replacing it.
- [ ] Later custom workflow adoption uses an explicit separate registration/catalog path rather than repository-local shadowing of built-in workflow identities.
- [ ] Shared-memory accelerators remain rebuildable derived-state helpers only.
- [ ] Later workflow extensibility does not add new privileged operations or weaken existing trust boundaries.
