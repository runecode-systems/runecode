## Summary
RuneCode adds explicit, auditable shared-workspace concurrency instead of relying on one-run-per-workspace indefinitely.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Workspace Concurrency Model.
- Conflict Detection + Isolation Rules.
- Runner, Broker, and TUI Integration.
- Fixtures + Recovery Cases.

## Why Now
This work remains scheduled for vNext, and keeping it on this canonical RuneContext change preserves direct roadmap-to-change traceability for later delivery and verification.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Workflow Concurrency v0 reviewable as a RuneContext-native change and removes the need for a second semantics rewrite later.
