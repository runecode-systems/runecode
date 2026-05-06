## Summary
Expand RuneCode's performance program beyond the MVP beta gate set to cover broader workflow-pack surfaces, git-gateway and broader project-substrate paths, heavier fixture tiers, and tuned cross-platform verification gates once the Linux-first beta baseline is already in place.

## Problem
`CHG-2026-053-9d2b-performance-baselines-verification-gates-v0` is now the MVP performance gate set for the first usable beta. That is the right first boundary, but it intentionally leaves valuable follow-on work outside the beta-critical lane:

- broader CHG-049 workflow-pack entry and mutation surfaces beyond the supported beta slice
- git-gateway performance checks when remote mutation is not part of the MVP hard gate
- larger fixture ladders and heavier extended-lane measurements that improve scale confidence but are not release-defining for the first beta
- tuned macOS and Windows numeric gates after platform-specific runtime support and noise characteristics are better understood

Without a separate post-MVP change, those deferred surfaces would either drift without an owner or keep getting pulled back into the MVP gate set in ways that slow beta without improving the truthfulness of the first release promise.

## Proposed Change
- Create one post-MVP performance-expansion lane that extends the MVP gate foundation from `CHG-2026-053-9d2b-performance-baselines-verification-gates-v0`.
- Reuse the reviewed performance-contract artifact family introduced by `CHG-053` rather than creating a second baseline or threshold declaration format.
- Reuse the `CHG-053` metric taxonomy across exact, absolute-budget, regression-budget, and hybrid-budget checks unless a later reviewed follow-up deliberately refines that taxonomy.
- Reuse the `CHG-053` statistical defaults as the starting point for broader post-MVP checks, including repeated-sample robust comparison for microbenchmarks, median plus `p95` plus explicit ceilings for latency metrics, fixed-window average/median plus max guardrails for CPU/process-behavior metrics, and exact checks for deterministic invariant counts.
- Reuse the `CHG-053` timing-boundary rule so broader checks still terminate on reviewed broker-owned or persisted milestones rather than advisory client-local or launcher-local heuristics when authoritative downstream surfaces exist.
- Add explicit measurement of broader CHG-049 first-party workflow-pack surfaces, including draft artifact generation, draft promote/apply, reviewed implementation-input-set validation or binding, direct CLI workflow triggering, repo-scoped admission control or idempotency, and fail-closed drift-triggered re-evaluation or recompilation costs when those surfaces are part of the supported product story.
- Add explicit performance checks for git-gateway and broader project-substrate paths when those surfaces become part of the supported user workflow.
- Expand from the reviewed MVP fixture inventory to larger broker-fixture ladders and heavier extended-Linux measurements that improve confidence beyond the first beta release-defining fixtures.
- Expand cross-platform performance verification from Linux-first smoke or trend collection toward tuned macOS and Windows numeric gates where feasible.
- Keep performance verification deterministic, CI-safe, and aligned with the same trust-boundary rules, broker-owned authority model, and topology-neutral architecture rule as correctness checks.
- Allow selected higher-noise metrics to be promoted to tighter authoritative Linux environments later if needed, but treat that as measurement-infrastructure refinement rather than a product-architecture fork.

## Why Now
Splitting this work out now preserves a clean contract:

- `CHG-053` owns the first MVP beta performance gates
- this change owns the broader post-MVP expansion

That lets the first beta ship with serious performance discipline while still preserving an explicit lane for the larger program that should follow.

## Assumptions
- The MVP gate set from `CHG-053` lands first and becomes the baseline for future expansion.
- Broader workflow-pack surfaces and project-substrate or git-gateway flows are important to measure, but they should not redefine the first beta gate set retroactively.
- The reviewed performance-contract artifact family, metric taxonomy, statistical defaults, and authoritative timing-boundary rules from `CHG-053` remain the starting contract for this expansion lane.
- Tuned macOS and Windows numeric gates should follow the relevant platform runtime and virtualization work rather than assuming Linux measurements transfer directly.
- Larger fixtures and heavier extended lanes are valuable for post-MVP confidence, but they should remain deterministic and CI-safe.

## Out of Scope
- Replacing the MVP performance gate set in `CHG-053`.
- Weakening Linux-first required gates for the supported beta surface.
- Introducing non-deterministic benchmarks, live external dependency checks, or CI flows that mutate repo state.

## Impact
This change keeps the broader performance program reviewable without making the first beta gate set too wide.

If completed, RuneCode will gain a cleaner post-MVP path for:

- broader workflow-pack performance coverage
- git-gateway and broader project-substrate performance coverage
- larger fixture tiers and heavier extended lanes
- tuned macOS and Windows numeric gates beyond the Linux-first baseline
- broader coverage that still reuses the same reviewed artifact model, metric semantics, statistical defaults, and authoritative timing-boundary rules established in `CHG-053`

That preserves the value of the MVP beta gates while keeping the larger performance program visible and intentional.
