# Design

## Overview
This change expands RuneCode's performance program beyond the MVP beta gate set defined in `CHG-2026-053-9d2b-performance-baselines-verification-gates-v0`.

The design goal is to preserve the MVP gate set as a stable release contract while adding broader post-MVP coverage for:

- broader first-party workflow-pack surfaces
- git-gateway and broader project-substrate paths
- larger broker and end-to-end fixture tiers
- tuned cross-platform gates beyond Linux-first numeric enforcement

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

## Broader Workflow-Pack Coverage
The post-MVP workflow-pack expansion should cover surfaces that are useful but were intentionally excluded from the MVP hard gate, such as:

- draft artifact generation when it is no longer merely the minimum supported workflow slice
- explicit draft promote/apply timing through the audited shared path
- reviewed implementation-input-set validation or binding costs for approved-change implementation entry
- direct CLI workflow-trigger latency for broader workflow families
- repo-scoped admission-control and idempotency timing across broader workflow-pack entry points
- fail-closed drift-triggered re-evaluation or recompilation costs across those broader surfaces

These checks should remain deterministic and should continue to measure the same broker-owned immutable `RunPlan` architecture rather than an alternate fast path.

## Git Gateway And Project-Substrate Coverage
This expansion lane should add explicit performance coverage for surfaces that are implemented and important, but not required in the first beta hard gate:

- git remote prepare
- execute against local bare remotes
- project substrate posture and preview flows
- local project substrate apply flows

These checks should remain local-only and deterministic where possible.

## Larger Fixture Ladders And Heavier Extended Lanes
The MVP gate set intentionally avoids overloading the first release with the heaviest fixture program. This change should add:

- larger broker-fixture ladders
- heavier workflow and ledger fixtures
- broader extended-Linux merge-queue or scheduled lanes
- wider drift and repair cost coverage where those surfaces are already supported

The goal is to increase confidence at scale without turning the first beta PR lane into a noisy bottleneck.

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

## CI Integration Shape
This change should favor:

- extended Linux lanes for heavier measurements
- macOS and Windows smoke or trend lanes first
- gradual promotion of stable flow families into numeric-gated cross-platform lanes only after noise and baseline quality are understood

Threshold storage and baseline governance should stay review-driven and check-only.

## Design Risks To Avoid
- Do not let broader expansion erode the usefulness of the MVP gate set.
- Do not add flaky or externally networked checks.
- Do not treat cross-platform numeric tuning as a substitute for actual platform readiness.
- Do not reward trust-path bypasses just because they improve a benchmark number.
