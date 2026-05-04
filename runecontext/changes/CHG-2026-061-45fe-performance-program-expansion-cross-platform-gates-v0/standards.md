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
This change exists to expand RuneCode's performance program after the MVP gate set is already in place.

That includes freezing the following clarifications for post-MVP work:

- broader workflow-pack surfaces can gain explicit budgets without widening the first beta gate set retroactively
- git-gateway and broader project-substrate performance checks should remain deterministic and local-first where feasible
- larger fixture ladders and heavier extended lanes are valuable, but should not destabilize the MVP PR gate
- broader macOS and Windows numeric tuning should remain explicit follow-on work rather than implied parity with Linux before the platform lanes are ready
- threshold updates and baseline refreshes still require explicit review rather than silent CI mutation

This change extends the MVP performance foundation from `CHG-053` rather than redefining RuneCode's trust or control-plane contracts.
