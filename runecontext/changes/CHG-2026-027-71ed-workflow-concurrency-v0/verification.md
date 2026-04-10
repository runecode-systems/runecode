# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its vNext roadmap bucket and title after migration.
- Confirm concurrency scope keys reuse shared logical workflow identities rather than retry/attempt-local IDs.
- Confirm partial blocking and lock waits are represented through coordination/detail surfaces instead of a new public lifecycle enum.
- Confirm approvals, gate attempts, gate evidence, and overrides remain run-bound under concurrency.

## Close Gate
Use the repository's standard verification flow before closing this change.
