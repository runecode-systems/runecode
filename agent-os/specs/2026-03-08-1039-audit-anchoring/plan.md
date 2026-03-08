# Audit Anchoring — Post-MVP

User-visible outcome: audit roots can be externally anchored (optional) to strengthen tamper-evidence beyond the local machine.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-audit-anchoring/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Anchoring Mechanisms

- Evaluate and implement at least one anchor target:
  - TPM PCR extension
  - RFC 3161 timestamping
  - lightweight witness service
- Explicit goal: strengthen integrity even if the local audit writer is compromised by providing an external receipt of the audit root.

## Task 3: Artifact + Audit Integration

- Store anchor receipts as artifacts.
- Record anchoring events in the audit chain.

## Task 4: Verification

- Extend verification to optionally validate anchor receipts.

## Acceptance Criteria

- Anchoring is optional, explicit, and produces verifiable receipts.
- Anchoring failure does not rewrite history; it flags runs and is audited.
