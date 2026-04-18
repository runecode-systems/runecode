# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its `v0.1.0-alpha.8` roadmap bucket and title after migration.
- Confirm dependency-fetch operations remain aligned with the shared gateway operation taxonomy rather than dependency-local outbound verbs.
- Confirm dependency-fetch audit fields remain aligned with the shared gateway audit evidence model.

## Close Gate
Use the repository's standard verification flow before closing this change.
