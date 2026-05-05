# Tasks

## Phase 1: Broader Workflow-Pack Coverage

- [ ] Reuse the reviewed performance-contract artifact family from `CHG-053` rather than defining a second baseline format for post-MVP checks.
- [ ] Reuse the `CHG-053` metric taxonomy and statistical defaults as the starting policy for broader checks unless later reviewed work explicitly refines them.
- [ ] Add deterministic checks for broader CHG-049 workflow-pack surfaces beyond the supported MVP beta slice.
- [ ] Add deterministic draft artifact-generation checks where those surfaces are part of the supported post-MVP product story.
- [ ] Add deterministic draft promote/apply checks for canonical RuneContext mutation through the shared audited path.
- [ ] Add deterministic reviewed implementation-input-set validation or binding checks for approved-change implementation entry.
- [ ] Add deterministic direct CLI workflow-trigger latency checks for broader workflow-pack entry points.
- [ ] Add deterministic repo-scoped admission-control and idempotency checks for broader workflow trigger paths.
- [ ] Add deterministic fail-closed re-evaluation or recompilation checks for project-context or approved-input drift on broader workflow-pack paths.
- [ ] Freeze authoritative timing boundaries for broader workflow-pack checks so they still terminate on reviewed broker-owned or persisted milestones.

## Phase 2: Git Gateway And Project-Substrate Expansion

- [ ] Add git gateway prepare performance checks against deterministic local fixture repos.
- [ ] Add git execute performance checks against deterministic local bare remotes.
- [ ] Add project-substrate posture and preview performance checks for deterministic fixture repos.
- [ ] Add local project-substrate apply performance checks for deterministic fixture repos.
- [ ] Apply the inherited metric taxonomy and authoritative timing-boundary rules to git-gateway and project-substrate checks.

## Phase 3: Larger Fixture Ladders And Heavier Lanes

- [ ] Add larger broker unary API fixture tiers beyond the MVP-supported buckets.
- [ ] Add larger broker watch-family fixture tiers beyond the MVP-supported buckets.
- [ ] Add heavier workflow execution fixtures for extended Linux lanes.
- [ ] Add heavier audit-ledger and verification fixtures for extended Linux lanes.
- [ ] Keep heavier lanes deterministic and suitable for merge-queue or scheduled execution.
- [ ] Treat larger fixture ladders as expansion from the reviewed MVP fixture inventory rather than as a separate fixture model.

## Phase 4: Cross-Platform Expansion

- [ ] Run the same flow families where feasible on macOS and Windows as smoke or trend collection after the relevant platform runtime work matures.
- [ ] Tune stable macOS numeric thresholds where fixture noise and platform behavior are understood.
- [ ] Tune stable Windows numeric thresholds where fixture noise and platform behavior are understood.
- [ ] Preserve Linux as the first authoritative numeric gate until the broader cross-platform program is genuinely stable.
- [ ] Promote selected higher-noise metrics to tighter authoritative Linux environments only if shared Linux CI proves insufficient, without changing metric identity or product architecture.

## Phase 5: Governance

- [ ] Document the promotion path from smoke or trend collection to numeric-gated cross-platform checks.
- [ ] Document the review process for broadening performance coverage without weakening the MVP gate set.
- [ ] Keep threshold updates and baseline refreshes review-driven and check-only.
- [ ] Keep threshold storage, metric semantics, statistical defaults, and timing-boundary rules aligned with the inherited `CHG-053` performance contract unless explicitly revised by reviewed follow-up work.

## Acceptance Criteria

- [ ] RuneCode has explicit post-MVP performance checks for broader workflow-pack surfaces beyond the first beta gate set.
- [ ] Git-gateway and broader project-substrate paths each have at least one deterministic CI-compatible performance check.
- [ ] Larger fixture ladders and heavier extended-Linux lanes exist without destabilizing the MVP beta PR gate.
- [ ] macOS and Windows run the same flow families where feasible, with tuned numeric gates added only where stable and meaningful.
- [ ] The broader performance program reuses the `CHG-053` performance-contract artifacts, metric taxonomy, statistical defaults, and authoritative timing-boundary rules unless explicitly revised through later reviewed work.
- [ ] The broader performance program remains aligned with the same trust-boundary and broker-owned authority model as the MVP gate set.
