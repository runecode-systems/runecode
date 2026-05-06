# Design

## Overview
This change expands RuneCode's performance program beyond the MVP beta gate set defined in `CHG-2026-053-9d2b-performance-baselines-verification-gates-v0`.

The design goal is to preserve the MVP gate set as a stable release contract while adding broader post-MVP coverage for:

- broader first-party workflow-pack surfaces
- git-gateway and broader project-substrate paths
- larger broker and end-to-end fixture tiers
- tuned cross-platform gates beyond Linux-first numeric enforcement

## Inherited Contract From CHG-053

This change extends the `CHG-053` performance foundation rather than redefining it.

That means the post-MVP expansion should continue to use:

- the reviewed performance-contract artifact family rather than a second baseline storage format
- the `CHG-053` metric taxonomy across exact, absolute-budget, regression-budget, and hybrid-budget checks unless explicitly refined by later reviewed work
- the reviewed `CHG-053` statistical defaults as the starting point for broader measurement classes
- the `CHG-053` timing-boundary rule that metrics terminate on reviewed broker-owned or persisted milestones whenever authoritative downstream surfaces exist
- the same topology-neutral architecture rule across constrained local devices and larger deployments

## Layer Boundary

### Layer 1: MVP Beta Gates
Owned by `CHG-053`:

- Linux-first numeric gates
- TUI idle and waiting behavior
- broker API and watch families for supported beta fixtures
- attach and resume
- supported workflow execution path
- launcher startup and truthful attestation path
- model-gateway, dependency-fetch, audit, protocol, and external-anchor checks

### Layer 2: Post-MVP Expansion
Owned by this change:

- broader CHG-049 workflow-pack surfaces beyond the supported beta slice
- git-gateway and broader project-substrate performance suites
- larger broker-fixture ladders and heavier extended-Linux measurements
- tuned macOS and Windows numeric gates where feasible

Layer 2 expands breadth and confidence. It does not introduce a second semantics model for thresholds, baselines, timing boundaries, or trust ownership.

## Broader Workflow-Pack Coverage
The post-MVP workflow-pack expansion should cover surfaces that are useful but were intentionally excluded from the MVP hard gate, such as:

- draft artifact generation when it is no longer merely the minimum supported workflow slice
- explicit draft promote/apply timing through the audited shared path
- reviewed implementation-input-set validation or binding costs for approved-change implementation entry
- direct CLI workflow-trigger latency for broader workflow families
- repo-scoped admission-control and idempotency timing across broader workflow-pack entry points
- fail-closed drift-triggered re-evaluation or recompilation costs across those broader surfaces

These checks should remain deterministic and should continue to measure the same broker-owned immutable `RunPlan` architecture rather than an alternate fast path.

Where broader workflow-pack checks add new timings, those timings should still terminate on reviewed broker-owned or persisted milestones rather than direct CLI-local proxies when authoritative downstream surfaces exist.

## Git Gateway And Project-Substrate Coverage
This expansion lane should add explicit performance coverage for surfaces that are implemented and important, but not required in the first beta hard gate:

- git remote prepare
- execute against local bare remotes
- project substrate posture and preview flows
- local project substrate apply flows

These checks should remain local-only and deterministic where possible.

Where git-gateway and project-substrate paths add exact counts, latency budgets, or regression budgets, they should use the same metric taxonomy and reviewed statistical defaults inherited from `CHG-053`.

## Larger Fixture Ladders And Heavier Extended Lanes
The MVP gate set intentionally avoids overloading the first release with the heaviest fixture program. This change should add:

- larger broker-fixture ladders
- heavier workflow and ledger fixtures
- broader extended-Linux merge-queue or scheduled lanes
- wider drift and repair cost coverage where those surfaces are already supported

The goal is to increase confidence at scale without turning the first beta PR lane into a noisy bottleneck.

This expansion should treat larger fixture ladders as a broadening of the reviewed MVP fixture inventory, not as permission to abandon the deterministic fixture discipline established by `CHG-053`.

## Cross-Platform Expansion
Linux remains the first authoritative numeric gate. This change is where cross-platform performance work becomes more ambitious.

### macOS
As macOS virtualization and runtime support mature, add the same flow families where feasible and tune numeric thresholds for:

- TUI startup and interaction
- broker local API and watch behavior
- supported workflow and launcher paths that are meaningful on macOS

### Windows
As Windows runtime support matures, add the same flow families where feasible and tune numeric thresholds for:

- TUI startup and interaction
- broker local API and watch behavior
- supported workflow and launcher paths that are meaningful on Windows

Cross-platform expansion must preserve one topology-neutral architecture rather than implying platform-local authority shortcuts.

It must also preserve one performance-contract model across platforms. Tuned thresholds and lane promotion may differ by environment, but artifact shape, metric semantics, timing-boundary discipline, and trust ownership should stay aligned.

## CI Integration Shape
This change should favor:

- extended Linux lanes for heavier measurements
- macOS and Windows smoke or trend lanes first
- gradual promotion of stable flow families into numeric-gated cross-platform lanes only after noise and baseline quality are understood

Selected higher-noise metrics may also be promoted to tighter authoritative Linux measurement environments if shared Linux CI proves too noisy, but that promotion should be treated as lane refinement rather than as a new product architecture or new metric identity.

Threshold storage and baseline governance should stay review-driven and check-only.

## Design Risks To Avoid
- Do not let broader expansion erode the usefulness of the MVP gate set.
- Do not add flaky or externally networked checks.
- Do not treat cross-platform numeric tuning as a substitute for actual platform readiness.
- Do not introduce a second baseline artifact family or a second metric semantics model for post-MVP checks.
- Do not terminate broader timings at advisory client-local milestones when reviewed broker-owned or persisted milestones exist downstream.
- Do not reward trust-path bypasses just because they improve a benchmark number.
