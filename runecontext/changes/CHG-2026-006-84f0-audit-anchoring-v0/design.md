# Design

## Overview
Add anchor receipts for signed `AuditSegmentSeal` commitments and integrate them with trusted local verification without changing the existing audit ledger substrate. MVP remains local-only, explicit, and brokered, with no network egress. Later external anchoring extends the same receipt and verification foundation rather than replacing it.

## Key Decisions
- Anchoring is an explicit post-seal action over the digest of a signed `AuditSegmentSeal` envelope, never over ad-hoc in-band segment-root events or mutable file-local markers.
- The canonical receipt object for anchoring is the shared `AuditReceipt` family with `audit_receipt_kind = anchor`; this change does not introduce a second top-level anchor-only receipt family.
- Anchor-specific semantics live in a typed anchor receipt payload bound through `receipt_payload_schema_id` and `receipt_payload`, so local and later external anchors share one envelope and one verifier path while still using target-specific typed witness data.
- Anchor receipts remain sidecar audit evidence rather than leaves inside the segment they attest, and the original sealed segment bytes plus original segment seal remain immutable after sealing.
- For locally produced segments, anchored posture is derived from valid anchor receipts over the original seal digest rather than from rewriting the segment file or replacing the original seal with a second mutable lifecycle seal.
- Failures are recorded; no history rewriting, segment-byte mutation, or ledger repair-by-resealing is permitted.
- MVP baseline anchoring is local-only and uses the purpose-scoped `audit_anchor` authority from `CHG-2026-005-cfd0-crypto-key-management-v0/` rather than a generic machine-key abstraction.
- Anchor receipts must be signed by a verifier record whose logical purpose is `audit_anchor`; allowed logical scopes remain topology-neutral and are limited to the node- or deployment-scoped authority model already defined by the crypto foundation.
- Approval, approval assurance, presence mode, and delivery channel remain distinct concepts. Policy may require approval for an anchoring action, but approval does not replace the user-presence-gated `audit_anchor` signing act and delivery channel must not become the trust primitive.
- MVP local anchoring should use real trusted-domain presence modes (`os_confirmation` or `hardware_touch`) by default. Explicit `passphrase`-based developer posture may remain available only when allowed by the crypto foundation and must be surfaced as degraded posture rather than treated as equivalent assurance.
- Verification keeps the existing dimensioned posture model (`integrity_status`, `anchoring_status`, `currently_degraded`, findings, reason codes) rather than inventing new top-level `verified_unanchored` or `verified_anchored` enums. Human-readable anchored/unanchored labels are derived from those stable verifier dimensions.
- Missing anchors are not a verification failure by default; they are degraded posture until later policy opts into requiring anchored posture for selected transitions or workflows.
- Invalid anchor receipts fail closed.
- External anchoring lives in `runecontext/changes/CHG-2026-025-5679-external-audit-anchoring-v0/` and must preserve this same receipt envelope, subject model, and fail-closed verification posture while adding explicit egress and target descriptors.

## Foundation Principles
- Preserve the hard trust boundary: the broker remains the only operator-facing control-plane surface; the TUI remains a strict client of broker-owned APIs; the runner remains uninvolved in trusted anchoring flow.
- Keep boundary-visible contracts typed and topology-neutral. No anchor object, approval object, or read-model field should depend on daemon-private file paths, local usernames, or host-specific handles.
- Preserve the audit ledger substrate from `CHG-2026-003`: `auditd` owns the authoritative ledger and verification path, while artifact-store copies remain optional export/review products rather than the primary audit record.
- Preserve local-first posture from the roadmap and standards: MVP local anchoring must not silently enable egress or bake external-target assumptions into the canonical anchor model.
- Optimize for later extensibility through addition, not replacement: external anchors, formal checks, and proof-oriented work should be able to extend the anchor payload families and policy posture without changing the receipt envelope, seal subject, or verifier dimension model.

## Canonical Anchor Receipt Model

### Shared Envelope
- Anchoring uses the existing `AuditReceipt` signed-object family with `audit_receipt_kind = anchor`.
- `subject_digest` remains the digest of the signed `AuditSegmentSeal` envelope being anchored.
- `subject_family` remains `audit_segment_seal`.
- The receipt envelope signature remains part of the canonical trust surface and is resolved through the shared verifier-record model from `CHG-2026-005-cfd0-crypto-key-management-v0/`.

### Typed Anchor Payload
- The receipt's anchor-specific payload should contain only semantics that are not already authoritatively derivable from the signed receipt envelope or the signed segment seal envelope.
- MVP anchor payload fields should include:
  - `anchor_kind`
  - `key_protection_posture`
  - `presence_mode`
  - optional `approval_assurance_level`
  - optional `approval_decision_digest`
  - typed witness or assertion data proving the local anchor act
- The anchor payload should not duplicate subject identity fields already carried by the signed receipt subject binding or signer identity fields already carried by the signed envelope signature.
- `anchor_kind` stays a typed machine-consumed discriminator, with `local_user_presence_signature` as the MVP baseline. Later target kinds remain follow-on work in `CHG-2026-025-5679-external-audit-anchoring-v0/`.

### Signer Constraints
- The verifier must treat the anchor receipt signer as part of the anchor contract, not merely as any valid receipt signer.
- `audit_receipt_kind = anchor` requires an `audit_anchor` logical-purpose signer.
- Unknown signer purpose, unsupported scope, or inconsistent anchor-posture evidence fails closed.

## MVP Local Anchoring Flow

### Ownership Model
- `broker` owns the operator-facing anchoring action surface and any policy/approval orchestration.
- `auditd` owns segment selection, authoritative receipt persistence, and verification refresh against the ledger.
- `secretsd` owns sign-request precondition checks and uses the `audit_anchor` authority to mint the anchor signature.
- `runecode-tui` remains a client of broker-owned anchoring APIs and projected posture surfaces only.

### Explicit Action Model
- MVP anchoring is explicit rather than silently coupled to every seal operation.
- The initial end-to-end secure path must stay healthy without mandatory anchoring, in line with the alpha.4 callout that hardening work must not displace the primary secure path.
- If automation is added later, it must call the same broker/auditd/secretsd path rather than introducing a second trust-divergent background-only route.

### Approval And Presence
- Policy decides whether an anchoring action requires approval.
- If approval is required, broker mints or resolves the shared signed approval objects before anchor signing proceeds.
- The actual anchor receipt is still a separate `audit_anchor` signing act whose sign-request preconditions record presence mode, key protection posture, and any linked approval decision context.
- Delivery channel metadata remains advisory UX context only.

## Artifact + Audit Integration
- The authoritative copy of anchor receipts lives in `auditd` sidecar evidence storage alongside other audit sidecars.
- Exported or review-oriented artifact copies are optional and must not become the primary trust source for audit anchoring in MVP.
- If artifact copies are later required for operator workflow, they should remain copies of the authoritative receipt and use a data-class choice that does not blur the distinction between the authoritative audit ledger and exported evidence products.
- Broker and TUI read models should project anchored posture and linked receipt identity from trusted verification and audit views rather than inventing a second anchoring contract.

## Verification Model
- Trusted Go verification remains authoritative.
- Verification of anchor receipts includes:
  - signed-envelope validity
  - receipt schema and typed anchor payload validation
  - anchor signer discovery and `audit_anchor` purpose enforcement
  - subject-family and subject-digest match against the verified segment seal digest
  - approval-link and posture consistency checks when anchor payload declares them
- Anchoring remains one posture dimension inside `AuditVerificationReport`.
- Missing anchors remain degraded by default.
- Invalid anchor receipts remain fail-closed anchoring failures.
- Human-readable anchored vs unanchored status shown in CLI, broker, and TUI remains derived from the machine-readable verifier report rather than becoming a second status taxonomy.

## Main Workstreams
- Canonical Anchor Receipt Model On Shared `AuditReceipt`
- Explicit Brokered Local Anchoring Flow (No Network Egress)
- Signer Authority, Approval, and User-Presence Binding
- Sidecar-Authoritative Audit Integration With Optional Export Copies
- Verification and Derived Anchoring Posture

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, typed contracts, or trusted state, the plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
