## Summary
RuneCode can run roles in a hardened container isolation mode when explicitly opted in, while clearly surfacing and auditing the reduced assurance.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Opt-In UX + Audit.
- Hardened Container Baseline.
- No Host Mounts + Artifact Movement.
- Policy Integration.

## Why Now
This work remains scheduled for v0.1.0-alpha.3, and keeping it on this canonical RuneContext change preserves direct roadmap-to-change traceability for later delivery and verification.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Container Backend v0 (Explicit Opt-In) reviewable as a RuneContext-native change and removes the need for a second semantics rewrite later.
