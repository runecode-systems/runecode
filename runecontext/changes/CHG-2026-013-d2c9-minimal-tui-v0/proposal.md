## Summary
Users can view runs, review diffs and artifacts, approve or deny high-risk actions, and inspect the audit timeline from a local TUI.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Bubble Tea App Skeleton.
- Core Screens (MVP).
- Local API Integration.
- Safety UX.

## Why Now
This work remains scheduled for v0.1.0-alpha.4, and keeping it on this canonical RuneContext change preserves direct roadmap-to-change traceability for later delivery and verification.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Minimal TUI v0 reviewable as a RuneContext-native change and removes the need for a second semantics rewrite later.
