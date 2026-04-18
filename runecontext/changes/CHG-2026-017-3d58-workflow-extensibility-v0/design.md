# Design

## Overview
Add generic authoring and review surfaces plus rebuildable shared-memory accelerators on top of the contract-first workflow substrate.

## Key Decisions
- The contract-first workflow-definition and binding substrate now lives in `CHG-2026-050-e3f8-workflow-definition-contract-binding-v0`.
- This change adds authoring, review, and accelerator capabilities on top of that substrate rather than redefining it.
- Future authoring adapters must normalize to the same RFC 8785 JCS canonical JSON bytes before validation and hashing that the contract-first substrate expects.
- Shared memory is a rebuildable accelerator for derived artifacts only; authoritative state remains in the run DB, artifact store, and audit trail.
- Generic authoring and accelerator work must preserve the shared workflow identity, executor, gate, approval, runner-binding, and git-composition contracts defined by the contract-first substrate.

## Authoring and Accelerator Posture

- Generic authoring surfaces should stay reviewable and deterministic; they must not become a plugin escape hatch around the contract-first workflow substrate.
- Shared-memory accelerators remain rebuildable caches for derived artifacts only and must not become authoritative workflow state.
- Authoring UX may prepare workflow-definition changes, but authoritative workflow truth still depends on canonical validated workflow definitions rather than client-local drafts.
- Generic authoring must preserve the same shared git-composition restrictions as the contract-first substrate; no process-local remote-mutation semantics are allowed.

## Main Workstreams
- Authoring + Review Surfaces
- Authoring Adapter Normalization
- Shared-Memory Accelerators
- Safe Adoption UX

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
