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
- Confirm any project-context-sensitive proof statement binds validated project-substrate snapshot identity rather than ambient repository assumptions.
- Confirm any runtime-sensitive proof statement binds the attested runtime identity seam from `CHG-2026-030-98b8-isolate-attestation-v0` rather than only pre-attestation launch assumptions or ambient platform-specific state.
- Confirm any audit-anchoring-sensitive proof statement binds canonical `AuditSegmentSeal` identity, authoritative anchor receipt identity, canonical target descriptor identity where relevant, and typed sidecar proof references rather than flattened summaries or exported-copy artifacts.

## Close Gate
Use the repository's standard verification flow before closing this change.
