## Summary
Establish RuneCode's first explicit performance baselines and CI verification gates across the full product surface: TUI idle and active behavior, broker local API request and watch latency, runner and workflow execution paths, launcher backend startup, model-gateway and secrets overhead, audit and protocol verification, git gateway paths, and end-to-end attach and resume flows.

## Problem
RuneCode currently has correctness-oriented checks but no durable performance gate. That leaves the product vulnerable to regressions that remain invisible until they become user-facing lag, noisy CPU usage, or slow attach and workflow behavior.

The immediate trigger for this change was a live TUI investigation driven by a user report that:

- system idle CPU before launch was roughly `1-3%`
- launching the TUI appeared to raise CPU toward `15-20%`
- key and navigation latency felt slow even on fresh launch
- lag appeared to worsen the longer the TUI stayed open

The investigation showed that RuneCode does not currently distinguish performance regimes clearly enough in planning or verification:

- empty-state TUI idle behavior
- active or waiting-session TUI behavior
- broker watch and projection costs
- local IPC request latency
- runner and workflow startup overhead
- backend startup and attach readiness
- provider, audit, protocol, and gateway overhead
- dependency-fetch cache miss, cache hit, and offline materialization overhead

Without deterministic fixtures, explicit thresholds, and CI enforcement, the project can regress in any of those areas without a visible review signal.

## Proposed Change
- Record the alpha.7 TUI bootstrap already implemented from this investigation: waiting-state activity now stays visibly marked without reusing the fast `running` repaint loop, and `cmd/runecode-tui` now has focused render/update benchmarks for shell view, watch apply, and palette entry construction.
- Capture the corrected performance investigation results as product planning guidance rather than leaving them as temporary terminal-session notes.
- Define RuneCode performance regimes explicitly, especially the difference between:
  - empty or quiescent local state
  - active or waiting session state
  - startup and attach latency
  - benchmarked render, watch, orchestration, and backend paths
- Introduce deterministic performance checks for all major RuneCode aspects, not just the TUI.
- Assign per-aspect thresholds that are suitable for CI, with Linux-first numeric gates and deterministic local fixtures or stubs instead of live external dependencies.
- Keep performance verification check-only and CI-safe so it remains compatible with `just ci` discipline and does not introduce silent writes or mutable benchmark artifacts during normal verification.
- Freeze a policy for baseline maintenance so future work can tighten thresholds intentionally instead of letting them drift implicitly.
- Include dependency-fetch and offline-cache performance as a first-class product regime, including cache miss, cache hit, miss coalescing, bounded concurrency, stream-to-CAS persistence, and broker-mediated offline dependency staging/materialization costs.
- Include explicit measurement of the refined CHG-050 workflow path, including definition validation/canonicalization, trusted compilation, compiled-plan persistence/load, and runner startup from immutable `RunPlan`.
- Preserve one topology-neutral performance program across constrained local hardware and larger deployments; tuning may differ, but performance work must not imply separate architecture paths or trust models.

## Why Now
RuneCode is approaching the first usable end-to-end Linux-first cut. That makes performance regressions more dangerous because users are no longer exercising isolated demos; they are exercising a connected product composed of the TUI, broker, runner, gateway, audit, and isolate layers.

The corrected TUI investigation also showed that the product needs a more precise narrative than "the TUI is slow":

- empty-state idle is already near the expected low CPU range when the broker store is truly isolated
- active or waiting work can still drive unacceptable sustained repaint cost and should be gated separately

Capturing that distinction now prevents future work from overfitting to the wrong problem statement and gives the project one durable performance-verification plan before more release-hardening work lands.

## Assumptions
- Linux CI will remain the first authoritative numeric-gate environment for the initial performance program.
- macOS and Windows should still execute the same flows where feasible, but their initial role is smoke, trend, and divergence detection until platform-specific baselines are tuned.
- Deterministic local fixtures, synthetic stores, local bare remotes, and stubbed provider backends are acceptable and preferred for CI gating.
- Network round-trip time to external providers is out of scope for hard CI gates; only RuneCode-added overhead should be measured under deterministic stubs.
- Performance checks must not weaken trust boundaries, bypass audit or policy, or replace canonical broker-owned state with client-local shortcuts.
- Performance verification is a product-quality concern and should remain part of the normal release and roadmap conversation, not a one-off local debugging artifact.

## Out of Scope
- Broad follow-on optimization work beyond the small alpha.7 waiting-state fix and focused benchmark coverage already landed.
- Publishing external provider SLA promises based on networked measurements.
- Replacing Bubble Tea, Lip Gloss, the broker architecture, or the runner architecture solely to satisfy this planning change.
- Treating one local developer machine profile as authoritative for every threshold.
- Adding CI steps that mutate repo state, rewrite baselines automatically, or depend on ambient external services.

## Impact
This change gives RuneCode one canonical planning surface for:

- the corrected TUI performance findings
- the distinction between empty-idle and active-state costs
- the profile-backed hot paths that deserve follow-on optimization work
- the deterministic benchmark and latency checks needed across the entire product
- the per-aspect thresholds and CI structure required to make performance a maintained contract rather than an anecdotal concern
- the explicit expectation that dependency-fetch performance must be measured on both cold-cache and warm-cache paths without weakening trust boundaries or buffering full dependency payloads in memory

It also freezes one durable product-level rule for future work:

- RuneCode performance should be evaluated per subsystem and per runtime regime, not as one vague "fast enough" claim for the whole product

- The same broker-owned workflow architecture should be measured and optimized across environments rather than replaced with different contract or authority paths for small-device versus scaled deployment shapes

The broader project-wide performance gates, real-child CPU harnesses, and subsystem-specific CI thresholds remain deferred to this later change and are not part of the alpha.7 implementation slice.
