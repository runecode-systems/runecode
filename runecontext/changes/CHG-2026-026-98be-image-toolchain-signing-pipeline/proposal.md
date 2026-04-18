## Summary
Isolate images and toolchains are signed and enforced at boot to reduce supply-chain risk.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Signing Key Hierarchy.
- Build + Publication Pipeline.
- Launcher Enforcement.
- Audit + Verification Integration.

## Why Now
This work now lands in `v0.1.0-alpha.9`, because isolate attestation and the first usable release both need signed runtime image and toolchain identity before measured provisioning can be treated as durable assurance.

Landing signed image and toolchain identity before the beta cut gives later attestation and verification work one stable runtime-image truth instead of a provisional foundation that would need to be rewritten.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Image/Toolchain Signing Pipeline reviewable as a RuneContext-native change and removes the need for a second semantics rewrite later.
