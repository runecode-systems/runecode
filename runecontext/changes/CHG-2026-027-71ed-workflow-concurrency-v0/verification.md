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
- Confirm shared-workspace concurrency composes with `CHG-2026-048-6b7a-session-execution-orchestration-v0` scoped blocking semantics instead of treating any single run wait as a workspace-global stop by default.
- Confirm approvals, gate attempts, gate evidence, and overrides remain run-bound under concurrency.
- Confirm concurrent runs may reuse broker-owned immutable dependency artifacts without promoting workspace-local caches or unpacked trees into authoritative dependency identity.
- Confirm dependency scope enablement or expansion approvals remain run- and scope-bound under concurrency rather than becoming workspace-global capability grants.
- Confirm validated project-substrate snapshot identity is not silently merged or ignored under concurrency.
- Confirm project-substrate drift or conflicting project-context bindings fail closed or surface explicit coordination/remediation posture.
- Confirm concurrency ownership and coordination remain broker-owned within the canonical repo-scoped product lifecycle rather than depending on client-local attach state, transport bindings, or workbench-local ownership heuristics.
- Confirm this change stays distinct from isolated implementation-track execution in `CHG-2026-051-4b9d-implementation-track-decomposition-git-worktree-execution-v0`.

## Close Gate
Use the repository's standard verification flow before closing this change.
