# Verification

## Planned Checks
- `just test`
- `go test ./internal/protocolschema`
- `cd runner && node --test scripts/protocol-fixtures.test.js`
- `cd runner && npm run boundary-check`

## Verification Notes
- Confirm the change distinguishes the audit plane from the verification plane and gives each a stable meaning.
- Confirm the design keeps canonical evidence authoritative and keeps derived indexes, search surfaces, dashboards, and watch views rebuildable.
- Confirm the foundation optimizes for inspectable evidence, deterministic verification, portable bundles, strong provenance, and explicit degraded posture.
- Confirm zero-knowledge proofs are described as an optional future privacy layer rather than the `v0` foundation.
- Confirm the change captures the full audience set: users, approvers, companies, auditors, incident response, external relying parties, fleet operators, privacy teams, and managed-service operators.
- Confirm the design includes signed receipts, append-only seal chains, runtime evidence, external anchoring, and verifier identity.
- Confirm the object-model split assigns index and record inclusion to `CHG-2026-056-8c75-audit-evidence-index-record-inclusion-v0`, preservation and bundle export to `CHG-2026-055-546a-verification-evidence-preservation-bundle-export-v0`, and coverage expansion to `CHG-2026-058-04e9-verification-coverage-expansion-v0`.
- Confirm the change makes degraded posture, denials, deferrals, overrides, and missing-evidence findings explicit rather than optional.
- Confirm the design preserves one architecture across constrained and scaled deployments and does not introduce a second truth surface or second authorization engine.
- Confirm proof-specific CLI, protocol, API, and dependency work is explicitly excluded from the foundation.

## Close Gate
Use the repository's standard verification flow before closing this change.
