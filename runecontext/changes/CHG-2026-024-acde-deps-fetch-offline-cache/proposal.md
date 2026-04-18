## Summary
RuneCode can fetch dependencies without giving workspace roles internet access.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Dependency Fetch Gateway Contract.
- Offline Cache Artifact Model.
- Policy + Audit Integration.

## Why Now
This work now lands in `v0.1.0-alpha.8`, because first-party implementation workflows need dependency material without granting workspace roles internet access.

Landing dependency fetch and offline cache before the first productive workflow pack keeps isolated implementation flows on the intended no-workspace-egress architecture instead of relying on later retrofits.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Deps Fetch + Offline Cache reviewable as a RuneContext-native change and removes the need for a second semantics rewrite later.
