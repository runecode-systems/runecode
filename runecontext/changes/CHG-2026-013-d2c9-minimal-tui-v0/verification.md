# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its v0.1.0-alpha.4 roadmap bucket and title after migration.
- Confirm the TUI plan distinguishes exact-action approvals from stage sign-off and explains supersession/staleness from typed approval data rather than scraped prose.
- Confirm approval UI notes keep `policy_reason_code`, `approval_trigger_code`, and system error states distinct.

## Close Gate
Use the repository's standard verification flow before closing this change.
