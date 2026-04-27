# Design

## Overview
Add generic authoring and review surfaces plus rebuildable shared-memory accelerators on top of the contract-first workflow substrate.

## Key Decisions
- The contract-first workflow-definition and binding substrate now lives in `CHG-2026-050-e3f8-workflow-definition-contract-binding-v0`.
- This change adds authoring, review, and accelerator capabilities on top of that substrate rather than redefining it.
- Generic authoring must target the refined split from CHG-050 rather than a single undifferentiated workflow object family:
  - `WorkflowDefinition` for workflow-facing selection and packaging
  - `ProcessDefinition` for executable graph structure
- Future authoring adapters must normalize to the same RFC 8785 JCS canonical JSON bytes before validation and hashing that the contract-first substrate expects.
- Shared memory is a rebuildable accelerator for derived artifacts only; authoritative state remains in the run DB, artifact store, and audit trail.
- Generic authoring and accelerator work must preserve the shared workflow identity, executor, gate, approval, runner-binding, and git-composition contracts defined by the contract-first substrate.
- Generic authoring and review surfaces must also preserve the shared validated project-substrate snapshot-binding model for project-context-sensitive workflows.
- Generic authoring must target the shared execution contract from `CHG-2026-048-6b7a-session-execution-orchestration-v0`, including broker-owned wait vocabulary and separate `approval_profile` and `autonomy_posture` inputs, rather than inventing authoring-local execution modes.
- Generic authoring may expose explicit implementation-track declarations that later feed `CHG-2026-051-4b9d-implementation-track-decomposition-git-worktree-execution-v0`, but execution planning and scheduling authority remain broker-owned rather than authoring-UI-owned.
- Generic authoring must preserve the CHG-050 `v0` DAG-only executable-graph posture and must not invent loops, cycles, or authoring-local re-entrant execution semantics.

## Authoring and Accelerator Posture

- Generic authoring surfaces should stay reviewable and deterministic; they must not become a plugin escape hatch around the contract-first workflow substrate.
- Shared-memory accelerators remain rebuildable caches for derived artifacts only and must not become authoritative workflow state.
- Authoring UX may prepare workflow-definition and process-definition changes, but authoritative workflow truth still depends on canonical validated definitions plus broker-owned signed compilation/selection binding rather than client-local drafts.
- Authoring UX may prepare track declarations or wait-related fields only through canonical typed schema surfaces; it must not invent local metadata that changes execution semantics.
- Generic authoring must preserve the same shared git-composition restrictions as the contract-first substrate; no process-local remote-mutation semantics are allowed.
- Custom workflow authoring must not invent alternate project discovery, init, adopt, upgrade, or project-context binding semantics.
- Custom workflow authoring must not create a second compilation or runtime-authority path around broker-owned immutable `RunPlan` execution.

## Main Workstreams
- Authoring + Review Surfaces
- Authoring Adapter Normalization
- Shared-Memory Accelerators
- Safe Adoption UX

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
