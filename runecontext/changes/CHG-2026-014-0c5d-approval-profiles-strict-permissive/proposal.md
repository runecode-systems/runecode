## Summary
RuneCode supports selectable human-in-the-loop approval profiles beyond MVP `moderate`, mapping ordinary actions to approval frequency and assurance without weakening core security invariants or the fixed exact-action hard floors for high-risk operations such as git remote mutation and external audit anchor submission.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Approval Profile Model (Post-MVP).
- Strict Profile Semantics.
- Permissive Profile Semantics.
- Policy + Runner + TUI Integration.
- Explicit hard-floor treatment for `git_remote_ops` and other exact-action remote mutation approvals.
- Explicit hard-floor treatment for external audit anchor submission when it uses the shared remote-state-mutation gateway class.
- Explicit preservation of the shared split between formal approval timing (`approval_profile`) and operator-guidance cadence (`autonomy_posture`).
- Explicit preservation of the shared dependency-fetch checkpoint model so later profile expansion does not accidentally reintroduce per-cache-miss approval semantics.
- Explicit compatibility with the first-party workflow-pack foundations from `CHG-2026-049-1d4e-first-party-runecontext-workflow-pack-v0`, where approval-profile timing must apply consistently across draft promote/apply and approved-change implementation without inventing workflow-local approval modes.

## Why Now
This work remains scheduled for v0.2, and keeping it on this canonical RuneContext change preserves direct roadmap-to-change traceability for later delivery and verification. The git-gateway foundation now freezes one especially important post-MVP rule that this change must inherit: approval profiles can tune ordinary approval timing, but they cannot batch away or soften exact final approval for remote mutation. `CHG-2026-048-6b7a-session-execution-orchestration-v0` also freezes that approval timing and operator-guidance cadence are separate controls, so this change must not turn profile selection into a proxy for autonomy posture. `CHG-2026-049-1d4e-first-party-runecontext-workflow-pack-v0` is the first concrete productive workflow consumer of that split, so later profile expansion must remain compatible with built-in draft promote/apply and approved-change implementation behavior rather than creating workflow-local approval semantics.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Approval Profiles (Strict/Permissive) reviewable as a RuneContext-native change, aligned with the reviewed exact-action approval foundation for git and other hard-floor operations, and removes the need for a second semantics rewrite later.

This now also keeps later profile expansion aligned with the clarified dependency-fetch foundation: profiles may tune ordinary approval timing, but they must do so using canonical dependency-fetch scope and action semantics rather than ambiguous "dependency install" or per-fetch approval language.

It also keeps later profile expansion aligned with the external audit anchoring foundation: profiles may not batch, ambiently pre-authorize, or relax exact-action approval for remote external anchor submissions that are bound to canonical target descriptor identity and canonical typed request hash.
