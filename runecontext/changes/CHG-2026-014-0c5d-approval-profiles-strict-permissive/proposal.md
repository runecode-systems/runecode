## Summary
RuneCode supports selectable human-in-the-loop approval profiles beyond MVP `moderate`, mapping ordinary actions to approval frequency and assurance without weakening core security invariants or the fixed exact-action hard floors for high-risk operations such as git remote mutation.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Approval Profile Model (Post-MVP).
- Strict Profile Semantics.
- Permissive Profile Semantics.
- Policy + Runner + TUI Integration.
- Explicit hard-floor treatment for `git_remote_ops` and other exact-action remote mutation approvals.

## Why Now
This work remains scheduled for v0.2, and keeping it on this canonical RuneContext change preserves direct roadmap-to-change traceability for later delivery and verification. The git-gateway foundation now freezes one especially important post-MVP rule that this change must inherit: approval profiles can tune ordinary approval timing, but they cannot batch away or soften exact final approval for remote mutation.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Approval Profiles (Strict/Permissive) reviewable as a RuneContext-native change, aligned with the reviewed exact-action approval foundation for git and other hard-floor operations, and removes the need for a second semantics rewrite later.
