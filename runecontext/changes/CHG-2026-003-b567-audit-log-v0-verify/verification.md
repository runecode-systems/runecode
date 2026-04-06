# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/protocolschema`
- `just test`

## Verification Notes
- Confirm the change defines the authoritative audit ledger as instance-global and `auditd`-owned, with rebuildable indexes rather than a second source of truth.
- Confirm `AuditEvent` and `AuditReceipt` are planned as unsigned payload families wrapped in detached `SignedObjectEnvelope` objects rather than inline-signature objects.
- Confirm the audit foundation keeps `SignedObjectEnvelope` single-signature and models additional attestations as separate signed objects or receipts.
- Confirm the canonical record digest is defined once and reused for chain links, Merkle leaves, receipts, segment references, and verification reports.
- Confirm `AuditSegmentSeal` is a separate signed object and that segment seals/receipts remain sidecar evidence rather than in-band segment events.
- Confirm the segment model uses an ordered Merkle construction plus exact raw-file hashing and explicit lifecycle states.
- Confirm the change replaces ad-hoc parallel `related_*_hashes` fields with typed refs plus shared scope/correlation blocks and signer-evidence refs.
- Confirm the event model binds trust-relevant manifest and bundle context with exact hashes rather than relying on version labels alone.
- Confirm verification output is machine-readable, separates integrity from degraded posture, and uses stable finding codes/severities.
- Confirm related changes `CHG-2026-004`, `CHG-2026-005`, `CHG-2026-006`, `CHG-2026-008`, `CHG-2026-013`, `CHG-2026-025`, `CHG-2026-027`, `CHG-2026-030`, and `CHG-2026-031` remain aligned on authoritative ledger ownership, anchoring targets, verifier identity, import/restore provenance, and audit API/TUI consumption.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the change still matches its `v0.1.0-alpha.2` roadmap bucket and title after refinement.

## Close Gate
Use the repository's standard verification flow before closing this change.
