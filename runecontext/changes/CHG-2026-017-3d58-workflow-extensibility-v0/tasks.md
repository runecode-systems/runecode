# Tasks

## Authoring and Review Surfaces

- [ ] Define generic authoring and review surfaces for workflow definitions on top of the contract-first substrate.
- [ ] Keep authoring flows deterministic and explicit rather than plugin-like or freeform.
- [ ] Ensure authoring and review surfaces do not become a second source of authoritative workflow truth.

## Authoring Adapter Normalization

- [ ] Normalize any authoring adapters to the same canonical workflow-definition format and hashing rules established by `CHG-2026-050-e3f8-workflow-definition-contract-binding-v0`.
- [ ] Keep machine validation deterministic and explicit.

## Shared-Memory Accelerators

- [ ] Define rebuildable shared-memory accelerators for derived artifacts only.
- [ ] Keep authoritative state in the run DB, artifact store, and audit trail.

## Safe Adoption UX

- [ ] Ensure generic workflow authoring continues to reuse the shared identity, executor, gate, approval, audit, and git-composition contracts from `CHG-2026-050-e3f8-workflow-definition-contract-binding-v0`.
- [ ] Ensure custom workflow authoring cannot mutate repository policy truth, ref allowlists, or repository-specific commit policy through local settings or untyped side channels.
- [ ] Ensure custom workflow authoring reuses the shared validated project-substrate snapshot-binding model where project context is relevant.
- [ ] Ensure custom workflow authoring cannot invent alternate project discovery, init, adopt, or upgrade semantics through local settings or untyped side channels.

## Acceptance Criteria

- [ ] Generic workflow authoring and review remain deterministic and build on the contract-first workflow substrate rather than replacing it.
- [ ] Shared-memory accelerators remain rebuildable derived-state helpers only.
- [ ] Later workflow extensibility does not add new privileged operations or weaken existing trust boundaries.
