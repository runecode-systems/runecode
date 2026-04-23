## Summary
RuneCode can migrate local broker IPC to protobuf without changing the logical protocol or local-only trust posture.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Proto Mapping for the Existing Logical Model.
- Local IPC Transport Requirements.
- Optional Local-Only gRPC Profile.
- Migration and Compatibility Rules.
- Explicit preservation of the repo-scoped product lifecycle and canonical `runecode` user surface so transport migration does not leak socket/service-manager identity into the product contract.

## Why Now
This work remains scheduled for v0.2, and keeping it on this canonical RuneContext change preserves direct roadmap-to-change traceability for later delivery and verification.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` defines the canonical repo-scoped product instance model and top-level `runecode` lifecycle surface; this change must preserve those logical semantics while changing only transport encoding and binding.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Local IPC Protobuf Transport v0 reviewable as a RuneContext-native change and removes the need for a second semantics rewrite later, while preserving the logical broker-owned product lifecycle and canonical RuneCode user surface above transport.
