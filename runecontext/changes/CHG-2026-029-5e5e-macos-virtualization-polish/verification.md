# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its v0.2 roadmap bucket and title after migration.
- Confirm macOS runtime and packaging realization preserve one repo-scoped product instance per authoritative repository root rather than turning platform runtime artifacts into product identity.
- Confirm broker-owned product lifecycle posture remains the operator-facing truth on macOS rather than local IPC reachability or packaging/bootstrap state.
- Confirm canonical `runecode` attach/start/status/stop/restart semantics remain unchanged above macOS-specific runtime and packaging realization details.

## Close Gate
Use the repository's standard verification flow before closing this change.
