# Tasks

## Anchor Receipt Object Model (MVP)

- [ ] Define a typed anchor receipt object stored as a first-class artifact:
  - `AuditAnchorReceipt` (schema-defined; hash-addressable)
  - includes: `{schema_id, schema_version, anchor_kind, subject_schema_id, subject_digest, audit_segment_id, audit_segment_seal_digest, audit_segment_root_hash, created_at, signer_key_id, signer_scope, key_protection_posture, presence_mode, approval_assurance_level, approval_decision_hash, receipt_payload}`
- [ ] Define `anchor_kind` values:
  - MVP baseline: `local_user_presence_signature`
  - later non-MVP target kinds live in `runecontext/changes/CHG-2026-025-5679-external-audit-anchoring-v0/`
- [ ] Define how receipts are referenced from sidecar audit evidence and derived audit views without embedding large payloads in the sealed segment.
- [ ] Use the shared signed-object/verifier contract from `runecontext/changes/CHG-2026-005-cfd0-crypto-key-management-v0/` for receipt signatures and signer discovery.

Parallelization: can be implemented in parallel with audit writer/verify so long as the receipt schema and audit event reference shape are agreed first (coordinate with `runecontext/specs/protocol-schema-bundle-v0.md` and `runecontext/changes/CHG-2026-003-b567-audit-log-v0-verify/`).

## MVP Local Anchoring (No Network Egress)

- [ ] Implement an MVP anchoring mode that requires explicit user presence to mint a receipt:
  - sign the `AuditSegmentSeal` commitment using the purpose-scoped `audit_anchor` authority with a user-assurance gate (see `runecontext/changes/CHG-2026-005-cfd0-crypto-key-management-v0/`)
  - store the receipt as sidecar audit evidence and, if needed, an exported artifact copy while recording anchoring state in derived audit views
- [ ] If anchoring requires explicit approval under policy, mint and consume the shared signed `ApprovalRequest` and `ApprovalDecision` objects before the receipt is produced.
- [ ] Missing anchors are permitted by default (verification distinguishes anchored vs unanchored); policies may optionally require anchors for specific workflows later.
- [ ] Failure semantics:
  - anchoring failure does not rewrite history
  - anchoring failure is recorded and must be visible in TUI/verification output as a degraded posture

Parallelization: can proceed in parallel with the crypto and audit subsystems; it depends on the machine signing key + user-presence hook and on audit log segmentation/root hash commitments.

## External Anchoring Follow-On Spec

- [ ] Post-MVP external anchor targets now live in `runecontext/changes/CHG-2026-025-5679-external-audit-anchoring-v0/`.

Parallelization: none for MVP implementation; later anchoring work should build on the MVP receipt model defined here.

## Artifact + Audit Integration

- [ ] Store anchor receipts as sidecar evidence and optional exported artifacts.
- [ ] Keep anchoring state linked to segment seals without requiring an in-band anchoring event inside the sealed segment.

Parallelization: can proceed in parallel with audit log indexing/retention so long as the event types and artifact references are stable.

## Verification

- [ ] Extend verification to validate anchor receipts when present.
- [ ] Verification output must distinguish:
  - locally verified but unanchored segments
  - anchored segments (receipt present and valid)
  - invalid anchors (fails closed)

Parallelization: implement alongside `audit-log-verify-v0` verifier work; avoid conflicts by finalizing receipt schema first.

## Acceptance Criteria

- [ ] Anchoring is optional, explicit, and produces verifiable receipts.
- [ ] MVP supports at least `local_user_presence_signature` anchoring without requiring any network egress.
- [ ] Verification reports anchored/unanchored status and fails closed on invalid receipts.
- [ ] Anchoring failure does not rewrite history; it flags runs and is audited.
