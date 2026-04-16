## Summary
Key security invariants are formally specified and continuously model-checked, reducing the chance of subtle privilege-escalation or routing bugs.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Define Invariants to Specify (MVP Scope).
- Write TLA+ Specification.
- CI Model Checking.
- Traceability.

## Why Now
This work now lands in `v0.1.0-alpha.5`, after the core workflow, policy, audit, and broker foundations are in place, so the highest-risk invariants can be frozen and model-checked before MVP. Keeping it on this canonical RuneContext change preserves direct roadmap-to-change traceability for later delivery and verification.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Formal Spec v0 (TLA+ + CI Model Checking) reviewable as a RuneContext-native change and removes the need for a second semantics rewrite later.
