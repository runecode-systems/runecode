## Summary
RuneCode adds generic workflow-authoring and review surfaces plus rebuildable shared-memory accelerators on top of the contract-first workflow substrate without changing the safety model.

## Problem
The original workflow-extensibility plan bundled two different scopes together: the contract-first workflow definition and binding substrate needed before the first productive workflow pack, and the later authoring and accelerator work that should remain additive.

Keeping both scopes together would either delay the first usable product cut or tempt the product into a special-case built-in workflow path that later generic extensibility would have to imitate.

## Proposed Change
- Generic workflow-authoring and review surfaces.
- Deterministic authoring adapters that normalize to the canonical workflow-definition and process-definition contracts.
- Shared-Memory Accelerators.
- Safe adoption UX for custom workflow definitions on top of the shared workflow-definition substrate.
- Explicit separate registration/catalog path for later custom workflows rather than repository-local override or shadowing of product-shipped built-in workflow identities.
- Where authoring targets draft-like workflows, preserve the artifact-first plus explicit promote/apply model from `CHG-2026-049-1d4e-first-party-runecontext-workflow-pack-v0` rather than collapsing draft generation and canonical mutation into one ambient local-edit side effect.
- Explicit reuse of the refined CHG-050 workflow substrate, including:
  - `WorkflowDefinition` as workflow-facing selection/packaging
  - `ProcessDefinition` as executable graph structure
  - broker-owned signed compilation/selection binding plus immutable `RunPlan` runtime authority
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

It also freezes that generic authoring extends the refined workflow substrate rather than reopening it:
- authoring may prepare `WorkflowDefinition` and `ProcessDefinition` content
- authoring may not replace broker-owned compilation/binding authority
- authoring may not reintroduce loops, plugin semantics, or a second runtime authority in place of CHG-050
- authoring may not override or shadow product-shipped built-in workflow identities from CHG-049
