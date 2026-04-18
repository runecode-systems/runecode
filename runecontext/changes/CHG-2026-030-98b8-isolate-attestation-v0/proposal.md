## Summary
RuneCode can upgrade MVP TOFU isolate binding to measured, attestable provisioning without changing the core audit contract.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Attestation Evidence Model.
- Launch, Verification, and Policy Integration.
- TUI + Audit Posture.
- Fixtures + Cross-Platform Considerations.

## Why Now
This work now lands in `v0.1.0-alpha.9`, because the first usable release should move from TOFU-only provisioning posture to measured attested provisioning without changing the core audit contract.

Landing attestation after signed runtime-image identity but before the beta cut keeps the assurance model cumulative and avoids treating TOFU as the long-term normal posture for the first usable release.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Isolate Attestation v0 reviewable as a RuneContext-native change and removes the need for a second semantics rewrite later.
