## Summary
macOS microVM reliability and UX are improved without changing the security model.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- HVF Reliability + UX.
- Optional Virtualization.framework Backend.
- Packaging + Permissions.
- Explicit preservation of the repo-scoped product instance model, broker-owned product lifecycle posture, and canonical `runecode` lifecycle surface across macOS-specific runtime and packaging realization.

## Why Now
This work remains scheduled for v0.2, and keeping it on this canonical RuneContext change preserves direct roadmap-to-change traceability for later delivery and verification.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` defines the canonical product-lifecycle model this platform polish work must preserve: one repo-scoped product instance per authoritative repository root, broker-owned lifecycle posture, and canonical `runecode` attach/start/status/stop/restart semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps macOS Virtualization Polish reviewable as a RuneContext-native change and removes the need for a second semantics rewrite later, while ensuring macOS-specific runtime and packaging improvements stay additive beneath the same logical RuneCode product contract.
