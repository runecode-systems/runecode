# Verification

## Planned Checks
- `just test`
- `go test ./internal/protocolschema`
- `cd runner && node --test scripts/protocol-fixtures.test.js`

## Verification Notes
- Confirm `AuditEvidenceSnapshot` is described as a preservation manifest rather than a substitute for evidence.
- Confirm `AuditEvidenceBundleManifest` captures included objects, roots, verifier identity, trust-root digests, and disclosure posture.
- Confirm bundle export is required to be streaming-friendly.
- Confirm offline or independent verification is possible without RuneCode's UI or internal database.
- Confirm the design includes explicit export profiles and selective-disclosure declarations.
- Confirm the design avoids default raw secret, raw prompt, or raw provider-payload export when digests and typed metadata are sufficient.
- Confirm the feature preserves enough evidence identity for retention, backfill, export, and future cross-machine work.
- Confirm tests include bundle completeness, large-export streaming behavior, selective disclosure, and offline verification.

## Close Gate
Use the repository's standard verification flow before closing this change.
