# Verification

## Executed Investigation Checks
- Built fresh investigation binaries for the product, broker, and TUI entrypoints.
- Ran focused `go test ./cmd/runecode-tui` profiling passes covering shell view and watch-heavy paths.
- Launched PTY-backed TUI sessions against broker local IPC using real `runecode-tui` child PID sampling.
- Re-ran the live measurement after correcting the store-isolation mistake so empty-state and active-state results were not conflated.
- Re-ran the live PTY-backed CPU sampling after the alpha.7 waiting-state fix using fresh isolated empty and deterministic waiting-session stores.

## Executed Investigation Results
- Confirmed the real `runecode-tui` child can be sampled directly rather than attributing CPU to the `script` PTY wrapper.
- Confirmed the first live high-CPU sample was against non-empty repo-scoped broker state rather than a truly empty isolated broker store.
- Confirmed the corrected empty-state isolated measurement stayed near `0.5-1.0%` CPU even after the TUI remained open for roughly a minute.
- Confirmed the earlier non-empty-state live sample climbed through `22.81%` and `61.92%` CPU while the shell reported active session state, making it a waiting or active-state sample rather than an empty-idle sample.
- Confirmed focused profiles point more strongly to render, wrap, ANSI, regex, and palette or surface allocation cost than to a single broker request hot spot.
- Confirmed the post-fix isolated empty-state rerun measured `0.20%` fresh CPU, `0.80%` mid CPU, and `0.80%` aged CPU for the real `runecode-tui` child.
- Confirmed the post-fix isolated waiting-state rerun measured `0.00%` fresh CPU, `1.00%` mid CPU, and `1.00%` aged CPU for the real `runecode-tui` child.
- Confirmed the strongest before/after improvement is the waiting-state path: the prior waiting-state sample reached `22.81%` mid and `61.92%` aged CPU, while the post-fix deterministic waiting sample stayed at `1.00%` mid and aged CPU.
- Confirmed the post-fix waiting transcript still rendered a `WAITING session=sess-manual-multiwait` marker, so the CPU improvement did not come from suppressing the operator-visible waiting cue.

## Planned Automated Checks
- `go test ./cmd/runecode-tui -bench 'Benchmark(ShellViewEmpty|ShellViewWaitingSession|ShellWatchApply|BuildPaletteEntries)' -benchmem`
- deterministic PTY-based TUI empty-idle CPU gate
- deterministic PTY-based TUI waiting-state CPU gate
- deterministic broker unary local API latency suite
- deterministic broker watch-family latency suite
- deterministic attach and resume latency checks
- deterministic supported-workflow and launcher performance suite
- deterministic dependency-fetch, audit, protocol, model-gateway, and external-anchor performance suites

## Verification Notes
- Confirm the change preserves the corrected distinction between empty-state and waiting-state TUI measurements.
- Confirm the change records the most important methodological correction: socket isolation is not broker-store isolation.
- Confirm the design captures the profile-backed render and allocation hot spots, not just the top-line CPU numbers.
- Confirm performance checks are proposed for the supported MVP beta surfaces rather than the full eventual product surface.
- Confirm the change defines a reviewed performance-contract artifact family separate from `runecontext/assurance/baseline.yaml`.
- Confirm the reviewed performance-contract artifact family lives under `runecontext/assurance/performance/` with a manifest, per-surface contract files, and optional repeated-sample baselines where needed.
- Confirm one trusted repo-local compare/enforce tool exists under `tools/` and does not rewrite baselines during normal CI.
- Confirm the design freezes a metric taxonomy across exact, absolute-budget, regression-budget, and hybrid-budget checks.
- Confirm every metric declares lane authority and activation state before enforcement.
- Confirm the design freezes the reviewed statistical defaults for repeated microbenchmarks, latency metrics, CPU/process-behavior metrics, and exact metrics.
- Confirm statistical constants are defined for sample counts, warmup windows, p95 eligibility, repeated CPU/process windows, and practical noise floors.
- Confirm repeated-sample regression checks use a practical noise-floor policy in addition to significance or interval-based comparison logic.
- Confirm authoritative timing boundaries declare `start_event`, `end_event`, `clock_source`, `evidence_source`, and `included_phases`, and terminate on reviewed broker-owned or persisted milestones rather than advisory launcher-local or client-local proxies when reviewed downstream authority surfaces exist.
- Confirm the initial reviewed fixture inventory uses stable fixture IDs before baselines are collected.
- Confirm thresholds declare `threshold_origin` as `product_budget`, `investigation_baseline`, `first_calibration`, or `temporary_guardrail`.
- Confirm the refined CHG-050 workflow path is measured explicitly, including validation or canonicalization, trusted compilation, compiled-plan persistence/load, and runner startup from immutable `RunPlan`.
- Confirm the supported CHG-049 workflow-pack beta slice is measured explicitly while broader workflow-pack surfaces are deferred.
- Confirm dependency-fetch and offline-cache have explicit cold-cache, warm-cache, miss-coalescing, and materialization checks.
- Confirm dependency-fetch performance checks preserve the reviewed stream-to-CAS and bounded-memory posture rather than rewarding trust-boundary shortcuts, and that bounded-buffer instrumentation exists in addition to coarse process-memory observation.
- Confirm external audit anchoring has explicit checks for prepare latency, execute-completed latency, execute-deferred handoff latency, deferred completion visibility, and receipt admission over an unchanged verified seal.
- Confirm external audit anchoring performance checks do not reward forbidden shortcuts such as network I/O under audit-ledger lock, bypassing authoritative verifier admission, or forcing full verifier replay as the only normal receipt-admission path.
- Confirm launcher cold and warm startup checks are defined in terms of the signed runtime-asset path.
- Confirm attestation cold and warm checks are defined in terms of the required attestation trust path.
- Confirm attestation performance contracts remain `contract_pending_dependency` until `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` lands.
- Confirm launcher and attach-ready performance checks do not reward bypassing attestation verification, replay checks, freshness checks, or attestation evidence persistence.
- Confirm external-audit-anchor performance contracts remain `contract_pending_dependency` until `CHG-2026-025-5679-external-audit-anchoring-v0` lands.
- Confirm the first required Linux lane is scoped to metrics stable enough for shared hosted Linux thresholds, while leaving room to promote selected higher-noise metrics later without changing metric identity or product architecture.
- Confirm the roadmap places this work under `v0.1.0-alpha.11`.
- Confirm broader workflow-pack expansion, git-gateway expansion, larger fixture ladders, and tuned macOS or Windows numeric gates are deferred to `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.
- Confirm the change keeps performance verification check-only and CI-safe.

## Close Gate
Use the repository's standard verification flow before closing this change.
