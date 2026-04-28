# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the change defines a broker-owned implementation-track model with stable track identity and dependency edges.
- Confirm reviewed implementation-input sets from CHG-049 remain the authoritative upstream implementation inputs rather than ambient repository planning state.
- Confirm explicit track declarations override inferred grouping.
- Confirm inferred decomposition becomes a broker-owned proposed execution-plan artifact rather than a hidden runtime heuristic.
- Confirm the proposed execution-plan artifact remains planning/review state and does not become a second runner-consumed runtime authority alongside CHG-050 immutable `RunPlan`.
- Confirm git worktrees are treated as the preferred isolation substrate for eligible low-coupling tracks rather than shared-workspace opportunism.
- Confirm worktree paths and local branch names remain implementation-private and do not become public object identity.
- Confirm dependency-fetch and offline-cache authority remain broker-owned and repo-scoped under worktree execution rather than drifting into per-worktree cache ownership.
- Confirm canonical dependency identity remains bound to reviewed dependency request and resolved-unit semantics rather than worktree paths, unpacked trees, or package-manager-local caches.
- Confirm pending operator input or approval blocks only the directly affected track and direct downstream dependent tracks.
- Confirm unrelated eligible tracks may continue only when active plan, dependency graph, broker policy, coordination state, and project-substrate posture all allow it.
- Confirm multiple pending waits may coexist and resolve independently.
- Confirm track execution reuses the canonical approval-profile and autonomy-posture split rather than minting a second approval authority.
- Confirm no autonomous path mints or substitutes for signed human approval decisions.
- Confirm dependency scope enablement or expansion remains on the shared approval-bearing checkpoint model rather than becoming a per-worktree approval surface.
- Confirm project-context-sensitive tracks bind to validated project-substrate snapshot digest and fail closed on incompatible drift.
- Confirm approved implementation-input drift or mutation-sensitive repository drift also fails closed or forces re-evaluation rather than heuristic continuation.
- Confirm canonical linkage among tracks, sessions, runs, approvals, artifacts, audit records, and project context remains broker-owned and explicit.
- Confirm this change remains additive over CHG-050: executable graph structure and scoped blocking semantics come from the shared workflow substrate, while actual later parallel/worktree execution behavior is introduced here rather than promised earlier.
- Confirm this change remains additive over the CHG-049 `v0` baseline of at most one mutation-bearing shared-workspace run per authoritative repository root rather than silently replacing that baseline.
- Confirm the roadmap and change text both place this feature in `vNext (Planned)`.

## Close Gate
Use the repository's standard verification flow before closing this change.
