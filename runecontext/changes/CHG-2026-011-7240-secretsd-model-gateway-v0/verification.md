# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/protocolschema`
- `just lint`
- `just test`

## Verification Notes
- Confirm this change remains a project-level tracker and does not drift back into feature-level duplication.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm child feature links for `CHG-2026-031-7a3c-secretsd-core-v0` and `CHG-2026-032-4d1f-model-gateway-v0` remain current.
- Confirm the child features now inherit one reviewed foundation for secret leases, model request boundaries, gateway identity, broker-projected posture, and quota semantics rather than redefining them independently.
- Confirm the parent text preserves the hard trust-boundary rules: no second credential cache, no combined auth/model/workspace role, no daemon-private user API becoming canonical, and no raw provider payloads becoming the control-plane contract source of truth.

## Close Gate
Use the repository's standard verification flow before closing this change.
