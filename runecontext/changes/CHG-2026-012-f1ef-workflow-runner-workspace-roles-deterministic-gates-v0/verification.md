# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm this change remains a project-level tracker and does not drift back into feature-level duplication.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm child feature links for `CHG-2026-033-6e7b-workflow-runner-durable-state-v0`, `CHG-2026-034-b2d4-workspace-roles-v0`, and `CHG-2026-035-c8e1-deterministic-gates-v0` remain current.
- Confirm the parent docs freeze one shared contract for:
  - broker-authoritative run truth versus runner advisory state
  - stable logical workflow identities versus attempt identities
  - shared public lifecycle vocabulary versus detailed coordination state
  - exact-action approvals versus stage sign-off
  - role kind versus executor class
  - typed gate identity plus typed gate evidence
  - event-style runner->broker writes plus broker-owned projection
- Confirm the parent docs do not leave room for child features to invent parallel status, approval, executor, or evidence vocabularies.

## Close Gate
Use the repository's standard verification flow before closing this change.
