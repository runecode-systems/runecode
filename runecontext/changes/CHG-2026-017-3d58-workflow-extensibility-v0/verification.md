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
- Confirm the change reuses the shared workflow identity and attempt model rather than inventing process-local scope identities.
- Confirm the change reuses the shared typed executor, gate, approval, and runner->broker checkpoint/result contracts.
- Confirm workflows that compose git remote mutation reuse the shared typed git request families and signed patch artifact contracts rather than process-local git payloads.
- Confirm custom workflows cannot mutate repository policy truth, ref allowlists, or repository-specific commit policy through local settings or untyped side channels.
- Confirm workflow-composed git remote mutation still routes through `git_remote_ops` exact-action approval with canonical repo, ref, artifact, and expected-result bindings.

## Close Gate
Use the repository's standard verification flow before closing this change.
