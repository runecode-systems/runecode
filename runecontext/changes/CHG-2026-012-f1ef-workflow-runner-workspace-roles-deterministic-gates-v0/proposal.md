## Summary
RuneCode can execute an end-to-end run where the scheduler proposes steps, policy authorizes them, workspace roles perform work offline, and deterministic gates produce evidence artifacts.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Workflow Runner Contract (Untrusted Scheduler).
- Workflow Extensibility Follow-On Spec.
- Runner Persistence Rules (MVP).
- Workspace Roles (MVP Set).
- Propose -> Validate -> Authorize -> Execute -> Attest Loop.
- Deterministic Gates (MVP).
- Minimal End-to-End Demo Run.

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
Keeps Workflow Runner + Workspace Roles + Deterministic Gates v0 reviewable as a RuneContext-native change and removes the need for a second semantics rewrite later.
