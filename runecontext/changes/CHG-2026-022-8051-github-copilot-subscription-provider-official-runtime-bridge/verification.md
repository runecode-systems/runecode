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
- Confirm provider setup, account-linking, and auth posture remain broker-owned typed flows rather than runtime-local authority.
- Confirm guided TUI setup and straightforward CLI setup remain thin clients of broker-owned setup APIs.
- Confirm long-lived auth material remains isolated to `secretsd` and downstream token delivery stays on the canonical lease boundary.

## Close Gate
Use the repository's standard verification flow before closing this change.
