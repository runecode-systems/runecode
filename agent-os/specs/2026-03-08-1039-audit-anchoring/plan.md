# Audit Anchoring v0

User-visible outcome: audit segment roots can be anchored with verifiable receipts, strengthening tamper-evidence beyond local write-time verification.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-audit-anchoring/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Anchor Receipt Object Model (MVP)

- Define a typed anchor receipt object stored as a first-class artifact:
  - `AuditAnchorReceipt` (schema-defined; hash-addressable)
  - includes: `{schema_id, schema_version, anchor_kind, audit_segment_id, audit_segment_root_hash, created_at, signer_key_id, signer_posture, receipt_payload}`
- Define `anchor_kind` values:
  - MVP baseline: `local_user_presence_signature`
  - reserved (post-MVP): `tpm_pcr`, `rfc3161`, `witness_service`, `transparency_log`
- Define how receipts are referenced from audit events without embedding large payloads in the audit log.

Parallelization: can be implemented in parallel with audit writer/verify so long as the receipt schema and audit event reference shape are agreed first (coordinate with `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/` and `agent-os/specs/2026-03-08-1039-audit-log-verify-v0/`).

## Task 3: MVP Local Anchoring (No Network Egress)

- Implement an MVP anchoring mode that requires explicit user presence to mint a receipt:
  - sign the segment root commitment using the per-machine signing key with a user-presence gate (see `agent-os/specs/2026-03-08-1039-crypto-key-mgmt-v0/`)
  - store the receipt as an artifact and record an anchoring audit event referencing the receipt hash
- Missing anchors are permitted by default (verification distinguishes anchored vs unanchored); policies may optionally require anchors for specific workflows later.
- Failure semantics:
  - anchoring failure does not rewrite history
  - anchoring failure is recorded and must be visible in TUI/verification output as a degraded posture

Parallelization: can proceed in parallel with the crypto and audit subsystems; it depends on the machine signing key + user-presence hook and on audit log segmentation/root hash commitments.

## Task 4: External Anchoring Targets (Post-MVP)

- Evaluate and optionally implement at least one external anchor target:
  - RFC 3161 timestamping
  - lightweight witness service
  - transparency log
- External anchoring requires an explicit egress model and must never silently enable network access.

Parallelization: can be designed in parallel with provider/gateway work; implementation should wait until gateway egress models are stable.

## Task 5: Artifact + Audit Integration

- Store anchor receipts as artifacts.
- Record anchoring events in the audit chain.

Parallelization: can proceed in parallel with audit log indexing/retention so long as the event types and artifact references are stable.

## Task 6: Verification

- Extend verification to validate anchor receipts when present.
- Verification output must distinguish:
  - locally verified but unanchored segments
  - anchored segments (receipt present and valid)
  - invalid anchors (fails closed)

Parallelization: implement alongside `audit-log-verify-v0` verifier work; avoid conflicts by finalizing receipt schema first.

## Acceptance Criteria

- Anchoring is optional, explicit, and produces verifiable receipts.
- MVP supports at least `local_user_presence_signature` anchoring without requiring any network egress.
- Verification reports anchored/unanchored status and fails closed on invalid receipts.
- Anchoring failure does not rewrite history; it flags runs and is audited.
