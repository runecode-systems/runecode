# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its `v0.1.0-alpha.10` roadmap bucket and title after migration.
- Confirm authenticated external anchor submissions align with the shared remote-state-mutation gateway class rather than an external-only outbound category.
- Confirm target identity uses typed exact-match semantics rather than raw URL-only policy.
- Confirm the change explicitly decides whether external anchor submission is an approved automated posture or an exact-action approval boundary per submission.
- Confirm authenticated target access, if any, remains lease-bound through the shared secrets model.
- Confirm audit evidence includes canonical target identity, anchoring subject identity, outbound payload or subject hash, bytes, timing, outcome, and relevant lease or policy bindings.
- Confirm project-context-sensitive anchored evidence reuses validated project-substrate snapshot identity rather than inventing a second project-context reference.

## Close Gate
Use the repository's standard verification flow before closing this change.
