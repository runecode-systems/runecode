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
- Confirm the change reflects the refined CHG-050 split between `WorkflowDefinition` selection/packaging, `ProcessDefinition` executable graph structure, and broker-owned immutable `RunPlan` runtime authority.
- Confirm later custom workflow adoption uses an explicit separate registration/catalog path rather than repository-local shadowing of product-shipped built-in workflow identities from CHG-049.
- Confirm generic authoring and review remain deterministic and do not become a second source of authoritative workflow truth.
- Confirm shared-memory accelerators remain derived-only and non-authoritative.
- Confirm generic authoring reuses the shared validated project-substrate snapshot-binding model where project context is relevant.
- Confirm custom authoring does not invent alternate project discovery, init, adopt, or upgrade semantics.
- Confirm generic authoring targets the shared execution vocabulary from `CHG-2026-048-6b7a-session-execution-orchestration-v0`, including separate `approval_profile` and `autonomy_posture` inputs.
- Confirm generic authoring preserves the CHG-050 `v0` DAG-only executable-graph posture and does not introduce loops, cycles, or authoring-local re-entrant runtime semantics.
- Confirm generic authoring does not create a second compilation or runtime-authority path around broker-owned signed compilation/selection binding and immutable `RunPlan` execution.
- Confirm any authoring that targets draft-like workflows preserves the CHG-049 artifact-first plus explicit promote/apply posture rather than inventing an ambient local-edit mutation lane.
- Confirm any explicit implementation-track authoring remains a canonical input to `CHG-2026-051-4b9d-implementation-track-decomposition-git-worktree-execution-v0` rather than becoming scheduler authority on its own.

## Close Gate
Use the repository's standard verification flow before closing this change.
