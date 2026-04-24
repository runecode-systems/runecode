# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its `v0.2` roadmap bucket and updated title after the split.
- Confirm the change now builds on `CHG-2026-050-e3f8-workflow-definition-contract-binding-v0` rather than redefining contract-first workflow semantics.
- Confirm generic authoring and review remain deterministic and do not become a second source of authoritative workflow truth.
- Confirm shared-memory accelerators remain derived-only and non-authoritative.
- Confirm generic authoring reuses the shared validated project-substrate snapshot-binding model where project context is relevant.
- Confirm custom authoring does not invent alternate project discovery, init, adopt, or upgrade semantics.
- Confirm generic authoring targets the shared execution vocabulary from `CHG-2026-048-6b7a-session-execution-orchestration-v0`, including separate `approval_profile` and `autonomy_posture` inputs.
- Confirm any explicit implementation-track authoring remains a canonical input to `CHG-2026-051-4b9d-implementation-track-decomposition-git-worktree-execution-v0` rather than becoming scheduler authority on its own.

## Close Gate
Use the repository's standard verification flow before closing this change.
