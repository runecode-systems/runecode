# Design

## Overview
Define typed external anchoring targets, receipt verification, and explicit egress controls for non-local audit anchoring while preserving the shared `AuditReceipt(kind=anchor)` envelope, `AuditSegmentSeal` subject model, sidecar-authoritative evidence posture, and existing verifier status dimensions established by `CHG-2026-006-84f0-audit-anchoring-v0/`.

## Key Decisions
- Later non-MVP anchor targets use typed descriptors and typed anchor receipt payloads on the shared `AuditReceipt(kind=anchor)` envelope rather than creating a second top-level external-only receipt family.
- External anchoring is explicit opt-in with a clear egress model.
- `AuditSegmentSeal` remains the canonical anchoring subject for external targets just as it does for local anchors.
- External anchor receipts remain authoritative as sidecar audit evidence first; artifact-store copies remain optional export/review products.
- Verification/reporting must distinguish valid external anchors from deferred, unavailable, or invalid states using the shared verifier posture model rather than a separate external-only status taxonomy.
- External anchor submissions that mutate remote target state should align with the shared gateway remote-state-mutation class rather than an ad hoc outbound category.
- Target identity, approval scope, lease scope, and audit evidence for external anchoring should use the same reviewed shared-gateway discipline now frozen for git remote mutation.

## Shared Gateway Alignment

- External anchoring should use typed target identity with exact matching rather than raw URL-only policy.
- If a target requires authenticated remote mutation, short-lived credentials must remain lease-bound through the shared secrets model rather than target-local secret handling.
- The change should decide explicitly whether a given external anchoring posture is:
  - an approved automated posture under signed manifests and allowlists
  - or an exact-action approval boundary for each outbound anchor submission
- Whatever posture is chosen, it must be expressed through the shared policy and approval vocabulary rather than a target-local approval model.
- Audit evidence should include canonical target identity, anchoring subject identity, outbound payload or subject hash, bytes, timing, outcome, and any lease or policy bindings needed to verify the outbound action.

## Main Workstreams
- Later Anchor Target Model
- Egress + Trust Boundary Model
- Receipt, Audit, and Verification Integration
- Fixtures + Adapter Conformance

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
