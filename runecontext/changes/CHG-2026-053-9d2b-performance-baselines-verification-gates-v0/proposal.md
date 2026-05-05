## Summary
Establish RuneCode's first explicit MVP beta performance baselines and CI verification gates for the supported beta surface: TUI idle and waiting behavior, broker local API request and watch latency, supported workflow execution, launcher startup with the truthful attestation path, model-gateway and secrets overhead, dependency-fetch and offline-cache overhead, audit and protocol verification, external audit anchoring, and end-to-end attach or resume flows.

## Problem
RuneCode currently has correctness-oriented checks but no durable performance gate. That leaves the first usable beta surface vulnerable to regressions that remain invisible until they become user-facing lag, noisy CPU usage, or slow attach and workflow behavior.

The immediate trigger for this change was a live TUI investigation driven by a user report that:

- system idle CPU before launch was roughly `1-3%`
- launching the TUI appeared to raise CPU toward `15-20%`
- key and navigation latency felt slow even on fresh launch
- lag appeared to worsen the longer the TUI stayed open

That investigation showed that RuneCode needs explicit performance regimes and deterministic verification for the actual MVP beta promise, especially:

- empty-state TUI idle behavior
- waiting-session TUI behavior
- attach and resume latency
- broker watch and projection costs
- local IPC request latency
- supported workflow startup and execution overhead
- launcher startup and attach readiness
- provider, audit, protocol, dependency-fetch, and external anchor overhead

At the same time, the repository now has a broader set of performance surfaces than the MVP beta actually needs to hard-gate. If those broader surfaces remain inside the first gate set, the project risks either delaying beta or watering the gates down until they stop being useful.

## Proposed Change
- Record the alpha.7 TUI bootstrap already implemented from this investigation: waiting-state activity now stays visibly marked without reusing the fast `running` repaint loop, and `cmd/runecode-tui` now has focused render and update benchmarks for shell view, watch apply, and palette entry construction.
- Capture the corrected performance investigation results as product planning guidance rather than leaving them as temporary terminal-session notes.
- Define RuneCode MVP performance regimes explicitly, especially the difference between:
  - empty or quiescent local state
  - active or waiting session state
  - startup and attach latency
  - benchmarked render, watch, orchestration, and backend paths
- Introduce deterministic performance checks for the supported beta surfaces only.
- Assign per-aspect thresholds that are suitable for CI, with Linux-first numeric gates and deterministic local fixtures or stubs instead of live external dependencies.
- Define one reviewed performance-contract artifact family separate from `runecontext/assurance/baseline.yaml` so project-substrate assurance posture and performance-governance posture remain distinct.
- Store that reviewed performance-contract artifact family under `runecontext/assurance/performance/`, with a manifest plus per-surface contract files and one trusted repo-local enforcement tool under `tools/` that never rewrites baselines during normal CI.
- Freeze one metric taxonomy for the first gate set so each measurement uses the right contract model instead of one vague "benchmark" bucket:
  - exact checks for exact counters and invariant counts
  - absolute budgets for user-visible experience ceilings
  - regression budgets for stable repeated microbenchmarks and hot paths
  - hybrid budgets where both explicit product ceilings and baseline-regression limits matter
- Freeze one lane and activation taxonomy so initial required gates are honest about measurement authority and dependency readiness:
  - `required_shared_linux` for stable required checks on shared hosted Linux
  - `required_tight_linux` for checks that are required but too noisy for shared hosted Linux
  - `informational_until_stable` for useful checks that need calibration before blocking
  - `contract_pending_dependency` for suites whose contracts can be authored before the underlying reviewed path lands
  - `extended` for heavier non-PR or post-MVP measurements
- Freeze the initial statistical defaults for the first implementation slice:
  - repeated-sample robust comparison for microbenchmarks
  - median plus `p95` plus explicit ceilings for latency metrics
  - fixed-window repeated sampling with average or median plus max ceilings for CPU and process-behavior metrics
  - exact comparison for deterministic event-count, payload-count, and duplicate-work metrics
  - practical noise-floor thresholds in addition to statistical significance for repeated regression checks
- Freeze initial statistical constants before implementation starts: repeated microbenchmarks use at least `10` PR samples and preferably `20` baseline-refresh samples; cheap local latency metrics use enough samples for meaningful `p95` gates, with `30` trials as the default target; heavier lifecycle metrics may use median plus max ceilings until they are cheap enough for meaningful percentile gates; CPU and process-behavior metrics use explicit warmup plus fixed observation windows.
- Freeze the rule that performance timing boundaries must terminate on reviewed broker-owned or persisted milestones rather than advisory launcher-local or client-local heuristics whenever authoritative downstream milestones exist.
- Require each performance contract to declare `start_event`, `end_event`, `clock_source`, `evidence_source`, `included_phases`, `threshold_origin`, and stable fixture identifiers before the gate can become required.
- Keep performance verification check-only and CI-safe so it remains compatible with `just ci` discipline and does not introduce silent writes or mutable benchmark artifacts during normal verification.
- Freeze a policy for baseline maintenance so future work can tighten thresholds intentionally instead of letting them drift implicitly.
- Include dependency-fetch and offline-cache performance as an MVP product regime, including cache miss, cache hit, miss coalescing, bounded concurrency, stream-to-CAS persistence, and broker-mediated offline dependency staging or materialization costs.
- Include explicit measurement of the refined CHG-050 workflow path, including definition validation or canonicalization, trusted compilation, compiled-plan persistence or load, and runner startup from immutable `RunPlan`.
- Include explicit measurement of the supported CHG-049 first-party workflow-pack beta slice rather than every broader workflow-pack surface.
- Include explicit measurement of the required attestation path for supported runtime startup and attach flows, including cold verification, warm verification-cache hits, replay and freshness checks, and persisted attestation-evidence handling.
- Include explicit measurement of the external audit anchoring path, including prepare latency, execute handoff latency, deferred completion handling, target-proof admission cost, and verifier behavior on unchanged verified seals.
- Freeze one small reviewed MVP fixture inventory per major surface for the first gate set rather than starting with broad ladder coverage; larger fixture ladders remain post-MVP expansion work.
- Assign stable fixture IDs for the first reviewed inventory before collecting baselines so future fixture expansion does not churn existing metric identity.
- Treat dependency-fetch memory posture as an explicit architectural contract, measured through both coarse process memory observations and reviewed internal bounded-buffer instrumentation so stream-to-CAS behavior is verified directly rather than inferred only from RSS.
- Keep the first required Linux performance lane compatible with shared CI where thresholds are conservative enough to remain deterministic, while leaving room to promote selected high-noise metrics to a tighter authoritative Linux environment later without changing product architecture or metric identity.
- Defer broader workflow-pack surfaces, git-gateway performance expansion, larger fixture ladders, and tuned macOS or Windows numeric gates to `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.

## Why Now
RuneCode is approaching the first usable end-to-end Linux-first cut. That makes performance regressions more dangerous because users are no longer exercising isolated demos; they are exercising a connected product composed of the TUI, broker, runner, gateway, audit, and isolate layers.

The corrected TUI investigation also showed that the product needs a more precise narrative than "the TUI is slow":

- empty-state idle is already near the expected low CPU range when the broker store is truly isolated
- waiting work can still drive unacceptable sustained repaint cost and should be gated separately

Capturing that distinction now prevents future work from overfitting to the wrong problem statement and gives the project one durable performance-verification plan for the MVP beta before more release-hardening work lands.

## Assumptions
- Linux CI will remain the first authoritative numeric-gate environment for the initial MVP performance program.
- Shared hosted Linux CI is acceptable for the first required numeric-gate slice where thresholds are deliberately conservative and metric noise is understood; the design may later promote selected high-noise metrics to a tighter Linux environment without redefining the product contract.
- Deterministic local fixtures, synthetic stores, stubbed provider backends, and stubbed external anchor targets are acceptable and preferred for CI gating.
- Network round-trip time to external providers is out of scope for hard CI gates; only RuneCode-added overhead should be measured under deterministic stubs.
- Performance checks must not weaken trust boundaries, bypass audit or policy, or replace canonical broker-owned state with client-local shortcuts.
- The supported beta workflow slice is the right first workflow gate set; broader workflow-pack entry families and post-MVP workflow surfaces should be measured later rather than broadening the first beta gate set prematurely.
- The first performance implementation slice should start with the reviewed statistical defaults captured by this change and tune them only after implementation and validation data shows that a given metric class needs different handling.
- The first implementation should treat threshold origins explicitly as `product_budget`, `investigation_baseline`, `first_calibration`, or `temporary_guardrail` so later reviewers can distinguish product promises from provisional calibration values.
- External audit anchoring and truthful post-handshake attestation performance suites should align explicitly with the underlying reviewed changes they measure rather than relying on sequencing folklore.
- Harnesses and performance contracts may be defined before dependent paths are complete, but gates must not become required until their `activation_state` is no longer `contract_pending_dependency` and the reviewed path exists.
- Broader macOS and Windows numeric performance gates should follow later platform tuning work rather than blocking Linux-first beta readiness.

## Out of Scope
- Broad follow-on optimization work beyond the small alpha.7 waiting-state fix and focused benchmark coverage already landed.
- Broad workflow-pack performance expansion beyond the supported beta slice.
- Git-gateway performance gating when that surface is not part of the MVP beta hard gate.
- Larger broker-fixture ladders and heavier extended-lane checks that are valuable but not release-defining for the first beta.
- Tuned macOS and Windows numeric gates.
- Publishing external provider SLA promises based on networked measurements.
- Replacing Bubble Tea, Lip Gloss, the broker architecture, or the runner architecture solely to satisfy this planning change.
- Treating one local developer machine profile as authoritative for every threshold.
- Adding CI steps that mutate repo state, rewrite baselines automatically, or depend on ambient external services.

## Impact
This change gives RuneCode one canonical planning surface for the first durable performance contract of the MVP beta:

- the corrected TUI performance findings
- the distinction between empty-idle and waiting-state costs
- the deterministic benchmark and latency checks needed across the supported beta surface
- the per-aspect thresholds and CI structure required to make performance a maintained contract rather than an anecdotal concern
- the explicit metric taxonomy, statistical defaults, and practical noise-floor policy for the first durable gate set
- the explicit rule that reviewed broker-owned or persisted milestones are authoritative timing boundaries for performance checks
- the explicit separation between project-substrate assurance baseline state and reviewed performance-contract artifacts
- the explicit lane-state, activation-state, fixture-ID, and threshold-provenance model needed to make the performance program executable without rewarding shortcuts or creating flaky gates
- the explicit expectation that dependency-fetch performance must be measured on both cold-cache and warm-cache paths without weakening trust boundaries or buffering full dependency payloads in memory
- the explicit expectation that the supported first-party workflow slice becomes a measurable product surface rather than invisible orchestration overhead
- the explicit expectation that launcher and attach-ready performance checks measure the reviewed signed-runtime plus required-attestation trust path rather than rewarding bypasses around attestation verification, replay or freshness enforcement, or attestation evidence persistence

It also freezes one durable product-level rule for future work:

- RuneCode performance should be evaluated per subsystem and per runtime regime, not as one vague "fast enough" claim for the whole product

The broader post-MVP performance expansion remains tracked separately in `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.
