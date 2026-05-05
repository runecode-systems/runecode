## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/local-product-lifecycle-and-attach-contract.md`
- `standards/global/project-substrate-contract-and-lifecycle.md`
- `standards/global/session-execution-contract-and-watch-families.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/runner-durable-state-and-replay.md`
- `standards/security/trusted-runtime-evidence-and-broker-projection.md`

## Resolution Notes
This change exists to make RuneCode MVP beta performance a maintained product contract rather than a one-off local debugging exercise.

That includes freezing the following clarifications for the first gate set:

- performance verification must remain deterministic, reviewable, and CI-safe
- performance checks must respect the same trust boundaries and broker-owned authority surfaces as correctness checks
- TUI empty-idle and waiting-state costs are distinct product regimes and must not be collapsed into one metric
- broker request latency, watch-family cost, runner startup, supported workflow execution, launcher startup, gateway overhead, audit verification, external audit anchoring, and attach or resume paths all need explicit budgets
- performance-contract artifacts remain separate from project-substrate assurance baseline state; `runecontext/assurance/baseline.yaml` is not the home for CHG-053 threshold declarations
- performance-contract artifacts live under `runecontext/assurance/performance/` and are enforced by a trusted check-only repo tool rather than rewritten by CI
- the first gate set uses an explicit metric taxonomy across exact, absolute-budget, regression-budget, and hybrid-budget checks rather than one generic benchmark bucket
- every metric needs reviewed lane authority, activation state, stable fixture identity, threshold provenance, and timing-boundary metadata before required enforcement
- timing boundaries must terminate on reviewed broker-owned or persisted milestones whenever those authoritative surfaces exist downstream in the product contract
- the first implementation slice uses reviewed statistical defaults per metric class rather than one universal statistics rule for every check
- the first implementation slice freezes sample-count, warmup, p95-eligibility, and practical-noise-floor constants before required gates are enforced
- the first gate set should start with one small reviewed fixture inventory per major surface, while larger fixture ladders remain explicit post-MVP expansion work
- contracts for attestation and external audit anchoring may be authored before their reviewed dependency paths land, but required enforcement must wait until those paths exist
- the supported first-party workflow-pack beta slice needs explicit budgets now, while broader workflow-pack surfaces should be expanded later under `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`
- Linux is the first authoritative numeric gate for this layer, while broader cross-platform tuning should remain explicit post-MVP work
- threshold updates and baseline refreshes require explicit review rather than silent CI mutation

This change builds on the existing broker, runner, TUI, lifecycle, and watch-family foundations rather than redefining those product contracts locally.
