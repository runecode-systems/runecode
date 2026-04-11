# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/protocolschema`
- `cd runner && npm run boundary-check`
- `just test`

## Verification Notes
- Confirm this change remains a project-level tracker and does not drift back into feature-level duplication.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm child feature links for `CHG-2026-033-6e7b-workflow-runner-durable-state-v0`, `CHG-2026-034-b2d4-workspace-roles-v0`, and `CHG-2026-035-c8e1-deterministic-gates-v0` remain current.
- Confirm the parent docs freeze one shared contract for:
  - immutable broker-compiled `RunPlan` plus superseding-plan semantics instead of in-place mutation
  - broker-authoritative run truth versus runner advisory state
  - stable logical workflow identities versus attempt identities
  - shared public lifecycle vocabulary versus detailed coordination state
  - exact-action approvals versus stage sign-off
  - role kind versus executor class
  - trusted executor registry versus runner read-only projection
  - typed gate identity plus typed gate evidence
  - event-style runner->broker writes plus broker-owned projection
- Confirm the parent docs treat `WorkflowDefinition` and `ProcessDefinition` as operational planning inputs, not long-term reserved placeholders.
- Confirm the parent docs keep the runner thin and explicitly reject runner-local planning, runner-local auth truth, ad hoc gate ordering, and forked executor semantics.
- Confirm the parent docs do not leave room for child features to invent parallel status, approval, executor, or evidence vocabularies.

## Close Gate
Use the repository's standard verification flow before closing this change.
