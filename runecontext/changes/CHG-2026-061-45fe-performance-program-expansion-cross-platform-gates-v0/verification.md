# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the roadmap places this change under `v0.2 (Post-MVP)`.
- Confirm `CHG-053` remains the MVP beta performance gate set and this change is explicitly additive over it.
- Confirm `CHG-060` remains the required beta product-smoke owner for project-substrate lifecycle, `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation`.
- Confirm the change explicitly reuses the `CHG-053` performance-contract artifact family rather than introducing a second baseline format.
- Confirm the change explicitly reuses the `CHG-053` metric taxonomy, statistical defaults, and authoritative timing-boundary rules as the starting post-MVP contract.
- Confirm the proposal captures broader performance coverage for and beyond the CHG-060 beta workflow loop, git-gateway publication paths, broader project-substrate fixture coverage, larger fixture ladders, and tuned cross-platform gates as the main deferred layer.
- Confirm the design keeps Linux as the first authoritative numeric gate while allowing broader macOS and Windows work to grow in a controlled way.
- Confirm larger fixture ladders are framed as an expansion of the reviewed MVP fixture inventory rather than a second fixture model.
- Confirm the tasks keep performance verification deterministic, CI-safe, and review-driven.
- Confirm the change does not weaken the MVP gate set by silently moving required beta checks out of `CHG-053` or CHG-060 product smokes out of CHG-060.

## Close Gate
Use the repository's standard verification flow before closing this change.
