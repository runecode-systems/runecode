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
- Confirmed the earlier non-empty-state live sample climbed through `22.81%` and `61.92%` CPU while the shell reported active session state, making it an active or waiting-state sample rather than an empty-idle sample.
- Confirmed focused profiles point more strongly to render, wrap, ANSI, regex, and palette or surface allocation cost than to a single broker request hot spot.
- Confirmed the post-fix isolated empty-state rerun measured `0.20%` fresh CPU, `0.80%` mid CPU, and `0.80%` aged CPU for the real `runecode-tui` child.
- Confirmed the post-fix isolated waiting-state rerun measured `0.00%` fresh CPU, `1.00%` mid CPU, and `1.00%` aged CPU for the real `runecode-tui` child.
- Confirmed the strongest before/after improvement is the waiting-state path: the prior waiting-state sample reached `22.81%` mid and `61.92%` aged CPU, while the post-fix deterministic waiting sample stayed at `1.00%` mid and aged CPU.
- Confirmed the post-fix waiting transcript still rendered a `WAITING session=sess-manual-multiwait` marker, so the CPU improvement did not come from suppressing the operator-visible waiting cue.

## Planned Automated Checks
- `go test ./cmd/runecode-tui -bench 'BenchmarkShell(View|Watch|BuildPaletteEntries)' -benchmem`
- deterministic PTY-based TUI empty-idle CPU gate
- deterministic PTY-based TUI waiting-state CPU gate
- deterministic broker unary local API latency suite
- deterministic broker watch-family latency suite
- runner boundary and protocol performance suite
- launcher startup and attach-ready performance suite
- dependency-fetch cache miss, cache hit, coalescing, and materialization performance suite
- audit, protocol, gateway, and project-substrate performance suites

## Verification Notes
- Confirm the change preserves the corrected distinction between empty-state and active-state TUI measurements.
- Confirm the change records the most important methodological correction: socket isolation is not broker-store isolation.
- Confirm the design captures the profile-backed render and allocation hot spots, not just the top-line CPU numbers.
- Confirm performance checks are proposed for all major RuneCode aspects rather than just the TUI.
- Confirm each major subsystem has an explicit threshold policy or regression budget.
- Confirm the refined CHG-050 workflow path is measured explicitly, including validation/canonicalization, trusted compilation, compiled-plan persistence/load, and runner startup from immutable `RunPlan`.
- Confirm the CHG-049 first-party workflow-pack path is measured explicitly, including draft artifact generation, draft promote/apply, implementation-input-set validation/binding, direct CLI triggering, repo-scoped admission control/idempotency, and fail-closed drift-triggered re-evaluation/recompile costs.
- Confirm dependency-fetch and offline-cache have explicit cold-cache, warm-cache, miss-coalescing, and materialization checks.
- Confirm dependency-fetch performance checks preserve the reviewed stream-to-CAS and bounded-memory posture rather than rewarding trust-boundary shortcuts.
- Confirm Linux remains the first authoritative numeric gate while other platforms still execute the same flow families where feasible.
- Confirm the change preserves one topology-neutral performance program across constrained local and larger deployments rather than implying separate architecture paths.
- Confirm the roadmap places this work under `v0.1.0-beta.1`.
- Confirm the change keeps performance verification check-only and CI-safe.

## Close Gate
Use the repository's standard verification flow before closing this change.
