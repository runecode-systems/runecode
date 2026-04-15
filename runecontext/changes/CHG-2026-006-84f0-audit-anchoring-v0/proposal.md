## Summary
Signed audit segment seals can be anchored with verifiable receipts, strengthening tamper-evidence beyond local verification while preserving the existing trust-boundary and no-history-rewrite model.

## Problem
The current planning record does not yet lock the implementation foundation strongly enough for later local hardening, external anchoring, formal verification, and proof-oriented follow-on work. In particular, the change must decide how anchor receipts fit the shared audit receipt model, which trusted authority owns anchor signatures, how anchoring interacts with approvals and user presence, and how anchoring state is represented without rewriting sealed history.

## Proposed Change
- Canonical Anchor Receipt Model On Shared `AuditReceipt`.
- MVP Local Anchoring Flow (Explicit, Brokered, No Network Egress).
- Signer Authority, Approval, and User-Presence Binding.
- Sidecar-Authoritative Audit Integration With Optional Export Copies.
- Verification and Derived Anchoring Posture.

## Why Now
This work remains scheduled for `v0.1.0-alpha.4`, after the local audit, policy, artifact, and primary isolated execution path exist. The roadmap's Alpha Implementation Callouts require hardening work like audit anchoring to build on the real brokered audit, policy, artifact, approval, and isolated-backend path without displacing it. Locking the object model and trust semantics now avoids a later protocol fork or second semantics rewrite.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- `CHG-2026-003-b567-audit-log-v0-verify/` remains the authoritative audit substrate: `AuditSegmentSeal` is already the canonical anchoring subject and sidecar receipts remain outside sealed segment bytes.
- `CHG-2026-005-cfd0-crypto-key-management-v0/` remains the authoritative signer and approval foundation: `audit_anchor` is the trusted logical authority for anchor signatures, while approval assurance, user presence, and delivery channel remain separate concepts.
- `CHG-2026-013-d2c9-minimal-tui-v0/` continues to consume broker-projected anchoring posture rather than introducing direct ledger access or TUI-private semantics.

## Out of Scope
- Runtime implementation of the feature in this planning update.
- External anchor targets, egress policies, and target-specific witness formats beyond the local-only MVP; these stay in `runecontext/changes/CHG-2026-025-5679-external-audit-anchoring-v0/`.
- Making anchoring mandatory for general workflow completion in MVP.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
This change package becomes the canonical decision record for a stronger anchoring foundation:

- one shared audit receipt envelope instead of parallel receipt families
- one immutable `AuditSegmentSeal` anchoring subject instead of ad-hoc root hashes or rewrite-prone lifecycle shortcuts
- one explicit brokered anchoring action path instead of multiple trust-divergent entry points
- one verifier posture model that cleanly composes with later external anchors, formal reasoning, and proof systems
