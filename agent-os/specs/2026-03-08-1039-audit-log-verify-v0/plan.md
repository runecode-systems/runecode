# Audit Log v0 + Verify

User-visible outcome: every action and decision is recorded in a tamper-evident, hash-chained audit log with verifiable signatures and a local verification command.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-audit-log-verify-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Audit Event Model

- Define the MVP audit event types and required fields:
  - previous event hash
  - event payload hash
  - signer identity (component or isolate)
  - manifest hash binding
  - timestamps and monotonic sequence
- Ordering semantics (MVP):
  - Define a per-signer strictly monotonic `seq` (do not require global ordering across signers).
  - Treat wall-clock timestamps as advisory metadata; verification must not rely on synchronized clocks to establish integrity.
  - Define verifier rules for gaps/duplicates/rollbacks in `seq`.
- Include explicit schema identifiers so verification can be performed across schema versions.
- Make audit events gateway-role aware (role identity + egress category), so network activity is attributable and enforceable:
  - model egress events (model-gateway): allowlist id, destination descriptor, bytes, timing, outcome
  - auth egress events (future auth-gateway): login/refresh lifecycle events (no secrets in logs)
  - git/web/deps egress events (post-MVP gateways): allowlist id, destination descriptor, bytes, timing, outcome
- Record secrets lease lifecycle events as first-class audit events (issuance/renewal/revocation), without logging secret values.
- Add event types required to harden against “audit writer compromise” (see `agent-os/specs/2026-03-08-1039-audit-anchoring/`):
  - audit segment root commitment events (segment id + root hash)
  - anchor receipt recorded events (receipt artifact hash + anchor kind)
- Add event types for isolate session/key provisioning posture:
  - session start event includes `{isolate_id, session_id, session_nonce, isolate_pubkey, provisioning_mode=tofu, image_digest, handshake_transcript_hash}`.

Parallelization: can be implemented in parallel with the schema bundle and crypto primitives, but the event envelope + canonicalization rules must be finalized first.

## Task 3: Append-Only Audit Writer

- Implement an append-only audit log writer process/role.
- Enforce schema validation and signature verification at write time.
- Store audit logs on encrypted-at-rest storage by default (e.g., inside the encrypted workspace volume).
- Record storage protection posture in audit metadata; do not silently fall back to plaintext.
- Expose a local-only health/readiness signal (consumable via the broker local API) for supervision and TUI status.
- Threat model note (MVP): if the audit writer itself is fully compromised, local write-time verification alone is not sufficient. MVP mitigates this by supporting anchoring receipts for segment roots (see `agent-os/specs/2026-03-08-1039-audit-anchoring/`).
  - If anchors are missing, verification reports the run as verified-but-unanchored (degraded posture).
  - If anchors are present but invalid, verification fails closed.

Parallelization: can be implemented in parallel with the artifact store (CAS) and verifier so long as the on-disk segment format and root-hash commitment rules are agreed.

## Task 4: Redaction Boundaries (Minimal)

- Define what is always redacted in the default operational view.
- Ensure secrets never cross trust boundaries by construction (not only post-hoc scrubbing):
  - use schema field classification metadata (`secret` fields are rejected/stripped at boundary)
  - prefer allowlists over heuristic redaction

Parallelization: can proceed in parallel with schema work; it depends on schema field classification metadata.

## Task 5: Verify Command

- Implement a deterministic verifier that checks:
  - hash chain integrity
  - signature validity
  - required event ordering invariants
- If anchor receipts are present, validate them and surface anchored vs unanchored status.
- Produce a machine-readable verification artifact (data class: `audit_verification_report`).
- Store the verification output as an artifact in the CAS and attach it to the run metadata so the TUI can show a clear "verified / not verified" status.

Parallelization: can be implemented in parallel with the audit writer once the event model and segment/root commitment format are defined.

## Task 6: Segmentation + Retention (Minimal)

- Define an audit log segmentation model that preserves verifiability:
  - segment the log (e.g., per run and/or size/time windows)
  - each segment has a recorded root hash (committed as an audit event)
  - segment root hashes are the anchoring target for receipts (see `agent-os/specs/2026-03-08-1039-audit-anchoring/`)
- Define minimal retention/archival rules (so audit does not grow without bound) without rewriting history.

Operational note (MVP): backup/restore and upgrades must preserve verifiability.
- Define minimal backup/restore guidance for audit segments + indexes (copy-only; no rewriting).
- Record restore/import events as audit events (verifier must be able to explain provenance).

Parallelization: can be implemented in parallel with retention/GC in the artifact store as long as “no history rewriting” is preserved.

## Acceptance Criteria

- A run produces an auditable sequence of events covering: proposals, validations, authorizations, executions, gate results, and approvals.
- Verification can be run locally and fails closed with clear errors.
- Audit logs can be segmented/archived without breaking verification.
- Verification output is storable/retained as a first-class artifact for later review.
- Verification output surfaces anchored vs unanchored posture when receipts are configured/present.
