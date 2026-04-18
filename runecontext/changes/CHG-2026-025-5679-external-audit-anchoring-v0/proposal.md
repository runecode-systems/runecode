## Summary
RuneCode can optionally anchor audit roots to external targets with explicit egress, typed receipts, and the same shared gateway, approval, lease, and audit-evidence discipline used for other high-risk remote mutation lanes.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Later Anchor Target Model.
- Egress + Trust Boundary Model.
- Receipt, Audit, and Verification Integration.
- Fixtures + Adapter Conformance.
- Explicit alignment with shared remote-state-mutation gateway semantics where external anchoring mutates remote target state.

## Why Now
This work remains scheduled for v0.2, and keeping it on this canonical RuneContext change preserves direct roadmap-to-change traceability for later delivery and verification. The git-gateway foundation now clarifies how other high-risk outbound lanes should talk about target identity, exact-action approval, lease scope, and shared audit proof, and external anchoring should inherit those decisions instead of drifting into an ad hoc egress model.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps External Audit Anchoring v0 reviewable as a RuneContext-native change, aligned with the reviewed shared gateway remote-mutation foundation, and removes the need for a second semantics rewrite later.
