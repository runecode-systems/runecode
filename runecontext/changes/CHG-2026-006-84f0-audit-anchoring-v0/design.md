# Design

## Overview
Add anchoring receipts for signed audit segment seals and integrate them with verification. MVP includes a local-only anchoring mode with no network egress. Later external-anchoring work is tracked separately.

## Key Decisions
- Anchoring is an explicit step and produces receipts over `AuditSegmentSeal` commitments rather than over ad-hoc in-band segment-root events.
- Failures are recorded; no history rewriting.
- MVP baseline anchoring is local-only and uses the purpose-scoped `audit_anchor` authority from `CHG-2026-005-cfd0-crypto-key-management-v0/` rather than a generic machine-key abstraction.
- Anchor receipts are signed objects under the shared detached-attestation contract and remain sidecar audit evidence rather than leaves inside the segment they attest.
- Any approval, assurance, or user-presence requirement for anchoring follows the shared signed approval model; delivery channel is advisory and must not become the trust primitive.
- Verification distinguishes `verified_unanchored` vs `verified_anchored`; missing anchors are not a verification failure by default.
- Invalid receipts fail closed.
- External anchoring lives in `runecontext/changes/CHG-2026-025-5679-external-audit-anchoring-v0/` and requires an explicit egress model.

## Main Workstreams
- Anchor Receipt Object Model (MVP)
- MVP Local Anchoring (No Network Egress)
- External Anchoring Follow-On Spec
- Artifact + Audit Integration
- Verification

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
