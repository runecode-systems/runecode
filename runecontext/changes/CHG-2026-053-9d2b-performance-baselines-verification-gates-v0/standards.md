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
This change exists to make RuneCode performance a maintained product contract rather than a one-off local debugging exercise.

That includes freezing the following clarifications for future work:

- performance verification must remain deterministic, reviewable, and CI-safe
- performance checks must respect the same trust boundaries and broker-owned authority surfaces as correctness checks
- TUI empty-idle and active or waiting-state costs are distinct product regimes and must not be collapsed into one metric
- broker request latency, watch-family cost, runner startup, workflow execution, launcher startup, gateway overhead, audit verification, and attach or resume paths all need explicit budgets
- first-party workflow-pack entry, draft artifact generation, explicit promote/apply, approved-input binding, admission control, and fail-closed re-evaluation paths also need explicit budgets once CHG-049 lands
- Linux is the first authoritative numeric gate, while other platforms should execute the same flow families and gain tuned thresholds over time
- threshold updates and baseline refreshes require explicit review rather than silent CI mutation

This change builds on the existing broker, runner, TUI, lifecycle, and watch-family foundations rather than redefining those product contracts locally.
