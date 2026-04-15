# Tasks

## Canonical Anchor Receipt Model On Shared `AuditReceipt`

- [x] Make the shared `AuditReceipt` family with `audit_receipt_kind = anchor` the canonical object model for audit anchoring.
- [x] Do not introduce a second top-level `AuditAnchorReceipt` family that would fork the receipt contract, verifier path, or broker/TUI read models.
- [x] Define a typed anchor receipt payload bound through `receipt_payload_schema_id` and `receipt_payload`.
- [x] Keep the anchor payload minimal and non-duplicative:
  - include `anchor_kind`
  - include `key_protection_posture`
  - include `presence_mode`
  - include optional `approval_assurance_level`
  - include optional `approval_decision_digest`
  - include typed witness data for the anchor act itself
  - do not duplicate subject identity, segment-root identity, or signer-key identity fields already authoritatively available from the signed receipt envelope or signed segment-seal envelope
- [x] Define `anchor_kind` values:
  - MVP baseline: `local_user_presence_signature`
  - later non-MVP target kinds live in `runecontext/changes/CHG-2026-025-5679-external-audit-anchoring-v0/`
- [x] Define how anchor receipts are referenced from sidecar audit evidence and derived audit views without embedding large payloads in sealed segment bytes.
- [x] Use the shared signed-object/verifier contract from `runecontext/changes/CHG-2026-005-cfd0-crypto-key-management-v0/` for receipt signatures, signer discovery, and signer-purpose enforcement.

Parallelization: can be implemented in parallel with audit writer/verify so long as the receipt schema and audit event reference shape are agreed first (coordinate with `runecontext/specs/protocol-schema-bundle-v0.md` and `runecontext/changes/CHG-2026-003-b567-audit-log-v0-verify/`).

## Explicit Brokered Local Anchoring Flow (No Network Egress)

- [x] Implement anchoring as an explicit broker-owned action path rather than as an implicit requirement on every seal operation.
- [x] Keep the public control-plane path split by responsibility:
  - broker owns operator-facing anchoring operations and policy/approval orchestration
  - auditd owns authoritative sidecar receipt persistence and verification refresh
  - secretsd owns sign-request precondition checks and the actual `audit_anchor` signature operation
  - TUI remains a strict client of broker-owned anchoring APIs
- [x] Implement an MVP local anchoring mode that requires explicit user presence to mint a receipt:
  - sign the `AuditSegmentSeal` commitment using the purpose-scoped `audit_anchor` authority
  - record sign-request preconditions including `key_protection_posture` and `presence_mode`
  - keep the authoritative receipt copy in sidecar audit evidence
  - optionally export a review copy only if a concrete consumer needs it
- [x] Keep missing anchors permitted by default for MVP so anchoring hardening does not displace the primary secure path; later policy may require anchored posture for specific transitions or workflow classes.
- [x] Failure semantics:
  - anchoring failure does not rewrite history
  - anchoring failure does not mutate sealed segment bytes or replace the original segment seal
  - anchoring failure is recorded and must be visible in broker/TUI verification posture as degraded or failed anchoring depending on cause

Parallelization: can proceed in parallel with the crypto and audit subsystems; it depends on the machine signing key + user-presence hook and on audit log segmentation/root hash commitments.

## Signer Authority, Approval, And User-Presence Binding

- [x] Require anchor receipts to be signed by verifier records whose logical purpose is `audit_anchor`.
- [x] Keep allowed anchor signer scopes topology-neutral and aligned with the crypto foundation (`node` or `deployment`), rather than baking host-local identity into the contract.
- [x] Fail closed when an anchor receipt is signed by the wrong logical purpose, unsupported scope, or otherwise inconsistent signer posture.
- [x] If policy requires approval for an anchoring action, mint and consume the shared signed `ApprovalRequest` and `ApprovalDecision` objects before the anchor receipt is produced.
- [x] Keep approval assurance, user presence, and delivery channel distinct:
  - approval authorizes the action when required by policy
  - the anchor signature remains a separate trusted signing act
  - delivery metadata is advisory and must not become the trust primitive
- [x] Treat `os_confirmation` and `hardware_touch` as the preferred MVP local presence modes.
- [x] Allow `passphrase` posture only when explicitly supported by the crypto foundation and surface it as degraded assurance rather than treating it as equivalent to hardware-backed or OS-keystore-backed confirmation.

Parallelization: can proceed in parallel with the crypto approval and sign-request work once shared signer-purpose and posture contracts are fixed.

## Sidecar-Authoritative Audit Integration With Optional Export Copies

- [x] Store anchor receipts as authoritative sidecar audit evidence first.
- [x] Keep any artifact-store copies explicitly optional and review/export oriented; the artifact store must not become the primary trust source for anchor receipts.
- [x] Keep anchoring state linked to the original segment seal digest without requiring an in-band anchoring event inside the sealed segment.
- [x] Preserve immutable sealed bytes and original seal identity after anchoring.
- [x] Represent anchored posture for locally produced segments as derived state from valid anchor receipts over the original seal, not as a segment-byte rewrite or replacement seal flow.
- [x] Keep broker and TUI audit surfaces aligned to the existing audit operational-view and verification-summary model rather than creating a second anchoring-specific read model.

Parallelization: can proceed in parallel with audit log indexing/retention so long as the event types and artifact references are stable.

## Verification And Derived Anchoring Posture

- [x] Extend authoritative verification to validate anchor receipts when present.
- [x] Verification must check:
  - signed-envelope validity
  - typed anchor payload validity
  - `audit_anchor` signer-purpose and allowed-scope enforcement
  - `audit_segment_seal` subject-family enforcement
  - subject-digest match against the verified segment-seal digest
  - approval-link and posture consistency when declared by the anchor payload
- [x] Keep anchoring inside the existing verifier dimension model rather than inventing a new top-level status taxonomy.
- [x] Verification output must distinguish through existing status dimensions and findings:
  - locally verified but unanchored segments (`anchoring_status = degraded` by default)
  - anchored segments with valid receipts (`anchoring_status = ok`)
  - invalid anchors (`anchoring_status = failed`, fail closed)
- [x] Keep broker/TUI anchored vs unanchored labels derived from machine-readable verifier output rather than from separately maintained prose or heuristics.

Parallelization: implement alongside `audit-log-verify-v0` verifier work; avoid conflicts by finalizing receipt schema first.

## External Anchoring Follow-On Alignment

- [x] Keep post-MVP external anchor targets in `runecontext/changes/CHG-2026-025-5679-external-audit-anchoring-v0/`.
- [x] Preserve the shared `AuditReceipt(kind=anchor)` envelope and `AuditSegmentSeal` subject model so later egress-enabled target work extends the MVP foundation instead of replacing it.

Parallelization: none for MVP implementation; later anchoring work should build on the receipt envelope, signer rules, and verifier posture model defined here.

## Acceptance Criteria

- [x] Anchoring is optional, explicit, and produces verifiable receipts.
- [x] MVP supports at least `local_user_presence_signature` anchoring without requiring any network egress.
- [x] `AuditReceipt(kind=anchor)` is the canonical anchor object model; no parallel top-level anchor receipt family is introduced.
- [x] Anchor receipts target the digest of the signed `AuditSegmentSeal` envelope and remain sidecar evidence rather than in-band segment leaves.
- [x] Anchor receipts are signed by the purpose-scoped `audit_anchor` authority under the shared verifier-record model.
- [x] Verification reports anchored/unanchored posture through the existing dimensioned verification model and fails closed on invalid anchor receipts.
- [x] Anchoring failure does not rewrite history, mutate sealed segment bytes, or replace the original segment seal; it is surfaced as auditable degraded or failed posture.
