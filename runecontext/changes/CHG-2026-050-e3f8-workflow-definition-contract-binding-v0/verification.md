# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm this change captures the contract-first workflow substrate split out from `CHG-2026-017`.
- Confirm the change reuses shared identity, executor, gate, approval, audit, and runner-to-broker contracts rather than inventing process-local variants.
- Confirm workflow-composed git remote mutation still routes through shared typed git request, patch artifact, repository identity, and exact-action approval semantics.
- Confirm generic authoring UX and shared-memory accelerators remain out of scope here.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.8`.

## Close Gate
Use the repository's standard verification flow before closing this change.
