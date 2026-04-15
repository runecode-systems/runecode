---
schema_version: 1
id: security/audit-anchoring-receipt-and-verification
title: Audit Anchoring Receipt And Verification
status: active
suggested_context_bundles:
    - go-control-plane
    - protocol-foundation
---

# Audit Anchoring Receipt And Verification

When trusted RuneCode services mint, persist, project, or verify audit anchoring evidence:

- Use shared `AuditReceipt` with `audit_receipt_kind = anchor` as the only canonical anchor receipt family; do not introduce a parallel top-level anchor-only receipt object
- Bind every anchor receipt to the digest of a signed `AuditSegmentSeal` envelope and keep `subject_family = audit_segment_seal`; do not anchor ad-hoc mutable segment-local markers or rewritten segment bytes
- Keep anchor-specific semantics in typed `receipt_payload` data so later local and external anchor kinds extend one receipt envelope rather than fork the verifier path
- Treat anchor receipts as authoritative audit sidecar evidence; exported artifact copies are optional review products and must not replace audit-owned authoritative copies
- Keep anchored posture derived from valid anchor receipts over the original seal digest; do not rewrite sealed segment bytes, reseal history, or replace the original seal identity after anchoring
- Require anchor receipt signers to use verifier records with logical purpose `audit_anchor`; allowed logical scopes are `node` or `deployment`, and ambiguous signer purpose or scope must fail closed
- Keep approval authorization, approval assurance, user presence, and delivery channel as distinct concepts; approval may authorize anchoring but does not replace the trusted anchor signing act
- Prefer real local presence modes such as `os_confirmation` or `hardware_touch`; `passphrase` posture is allowed only behind explicit reviewed support and must surface as degraded assurance rather than equivalent assurance
- Keep the broker as the only operator-facing anchoring action surface, `auditd` as the authoritative receipt persistence and verification owner, and `secretsd` as the trusted signing and precondition owner
- Keep missing anchors as degraded posture unless stricter policy explicitly requires anchored posture for a given workflow or transition
- Fail closed on invalid anchor receipts, mismatched subject digests, invalid typed payloads, inconsistent approval linkage, or unsupported signer posture
- Keep operator-visible anchored and unanchored labels derived from machine-readable verification posture rather than a second manually maintained status taxonomy
