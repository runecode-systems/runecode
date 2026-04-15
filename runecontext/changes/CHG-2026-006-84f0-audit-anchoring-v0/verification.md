# Verification

## Checks Run
- `go test ./internal/auditd ./internal/secretsd ./internal/brokerapi -run AuditAnchor -count=1`
- `rg "CHG-2026-006-84f0-audit-anchoring-v0|audit anchoring|alpha\.4|v0\.1\.0-alpha\.4" runecontext`
- `rg "\.runecode-secretsd/state\.json|\.runecode-secretsd|state\.json"`

## Verification Notes
- The change remains aligned with `runecontext/project/roadmap.md` (`v0.1.0-alpha.4`) and keeps anchoring as follow-on hardening that does not displace the primary secure path.
- Implemented code paths confirm the canonical model is shared `AuditReceipt(kind=anchor)` over `AuditSegmentSeal` digest with typed anchor payload (`receipt_payload_schema_id = runecode.protocol.audit.receipt.anchor.v0`), not a parallel top-level receipt family.
- Ownership split is implemented as designed: broker action API (`audit_anchor_segment`), `auditd` sidecar-authoritative receipt persistence + verification refresh, and `secretsd` `audit_anchor` signing preconditions/signature.
- Verification remains in the existing dimensioned model (`anchoring_status` in `AuditVerificationReport`) and handles unanchored/degraded, anchored/ok, and invalid/failed posture without segment-byte rewrite.
- Optional export-copy behavior is present (`audit_receipt_export_copy`) while authoritative trust remains sidecar evidence.
- `cmd/runecode-broker/.runecode-secretsd/state.json` was removed as an accidental runtime artifact; no code/docs references require that path.

## Close Gate
Use the repository's standard verification flow before closing this change.
