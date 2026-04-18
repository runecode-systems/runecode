## Summary
RuneCode supports schema-validated custom workflows and rebuildable shared-memory accelerators without changing the safety model, including preserving shared typed git-gateway contracts instead of allowing process-local remote mutation semantics.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- `ProcessDefinition` Contract.
- Validation + Canonicalization.
- Shared-Memory Accelerators.
- Policy, Approval, and Audit Binding.
- Authoring + UX Surfaces.
- Explicit reuse of shared typed git request, patch artifact, and exact-approval contracts when workflows compose git remote mutation.

## Why Now
This work remains scheduled for v0.2, and keeping it on this canonical RuneContext change preserves direct roadmap-to-change traceability for later delivery and verification.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Workflow Extensibility v0 reviewable as a RuneContext-native change, aligned with the reviewed git-gateway authority model, and avoids a later rewrite where custom workflows accidentally become a side door around typed remote mutation, repository policy, or exact-action approval rules.
