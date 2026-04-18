# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/protocolschema`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its v0.2 roadmap bucket and title after migration.
- Confirm the bridge lane now explicitly inherits the canonical model boundary and lease-based token handoff rather than defining bridge-local substitutes.
- Confirm destination identity, quota semantics, and operator posture remain shared and broker-projected rather than bridge-local.
- Confirm provider setup, account-linking, and auth posture remain broker-owned rather than bridge-runtime-local authority.
- Confirm TUI and CLI provider setup flows remain thin adapters over broker-owned typed setup APIs.

## Close Gate
Use the repository's standard verification flow before closing this change.
