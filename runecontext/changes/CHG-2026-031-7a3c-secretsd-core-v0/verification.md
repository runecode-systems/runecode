# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/protocolschema`
- `just test`

## Verification Notes
- Confirm the split preserves `secretsd` core requirements from the prior combined change.
- Confirm downstream provider and gateway changes can reference this feature as the reusable secret-management boundary.
- Confirm the feature defines a typed reusable lease contract rather than only action intent.
- Confirm secret material still cannot flow through env vars, CLI args, logs, or ordinary boundary-visible protocol objects.
- Confirm restart-time handling preserves lease and revocation safety fail closed.
- Confirm any user-facing or operator-facing secrets posture remains broker-projected rather than becoming a separate daemon user API.

## Close Gate
Use the repository's standard verification flow before closing this change.
