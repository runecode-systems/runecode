# Audit Log v0 + Verify

User-visible outcome: every action and decision is recorded in a tamper-evident, hash-chained audit log with verifiable signatures and a local verification command.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-audit-log-verify-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Audit Event Model

- Define the MVP audit event types and required fields:
  - previous event hash
  - event payload hash
  - signer identity (component or isolate)
  - manifest hash binding
  - timestamps and monotonic sequence
- Include explicit schema identifiers so verification can be performed across schema versions.

## Task 3: Append-Only Audit Writer

- Implement an append-only audit log writer process/role.
- Enforce schema validation and signature verification at write time.
- Store audit logs on encrypted-at-rest storage by default (e.g., inside the encrypted workspace volume).
- Record storage protection posture in audit metadata; do not silently fall back to plaintext.
- Threat model note (MVP): if the audit writer itself is fully compromised, local write-time verification is not sufficient. Mitigations belong in audit anchoring/witnessing (post-MVP).

## Task 4: Redaction Boundaries (Minimal)

- Define what is always redacted in the default operational view.
- Ensure secrets never cross trust boundaries by construction (not only post-hoc scrubbing):
  - use schema field classification metadata (`secret` fields are rejected/stripped at boundary)
  - prefer allowlists over heuristic redaction

## Task 5: Verify Command

- Implement a deterministic verifier that checks:
  - hash chain integrity
  - signature validity
  - required event ordering invariants
- Produce a machine-readable verification artifact.

## Task 6: Segmentation + Retention (Minimal)

- Define an audit log segmentation model that preserves verifiability:
  - segment the log (e.g., per run and/or size/time windows)
  - each segment has a recorded root hash
- Define minimal retention/archival rules (so audit does not grow without bound) without rewriting history.

## Acceptance Criteria

- A run produces an auditable sequence of events covering: proposals, validations, authorizations, executions, gate results, and approvals.
- Verification can be run locally and fails closed with clear errors.
- Audit logs can be segmented/archived without breaking verification.
