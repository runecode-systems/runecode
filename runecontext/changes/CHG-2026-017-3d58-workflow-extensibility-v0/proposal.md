## Summary
RuneCode adds generic workflow-authoring and review surfaces plus rebuildable shared-memory accelerators on top of the contract-first workflow substrate without changing the safety model.

## Problem
The original workflow-extensibility plan bundled two different scopes together: the contract-first workflow definition and binding substrate needed before the first productive workflow pack, and the later authoring and accelerator work that should remain additive.

Keeping both scopes together would either delay the first usable product cut or tempt the product into a special-case built-in workflow path that later generic extensibility would have to imitate.

## Proposed Change
- Generic workflow-authoring and review surfaces.
- Deterministic authoring adapters that normalize to the canonical workflow-definition contract.
- Shared-Memory Accelerators.
- Safe adoption UX for custom workflow definitions on top of the shared workflow-definition substrate.
- Explicit reuse of the shared workflow-definition, git request, patch artifact, and exact-approval contracts defined elsewhere.
- Authoring surfaces may prepare explicit implementation-track declarations for `CHG-2026-051-4b9d-implementation-track-decomposition-git-worktree-execution-v0` and definitions that target `CHG-2026-048-6b7a-session-execution-orchestration-v0`, but they do not become execution-planning or scheduler authority.

## Why Now
This work remains scheduled for `v0.2`, because generic workflow authoring and rebuildable accelerators are additive once the contract-first substrate and first-party productive workflows are already in place.

Keeping this change post-MVP after the split avoids coupling the first usable product cut to authoring polish and accelerator work that should extend the foundation rather than define it.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps later workflow extensibility reviewable as a RuneContext-native change focused on generic authoring and accelerators, while the contract-first substrate now lands separately and earlier.
