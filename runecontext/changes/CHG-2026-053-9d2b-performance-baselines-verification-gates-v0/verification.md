# Verification

## Executed Investigation Checks
- Built fresh investigation binaries for the product, broker, and TUI entrypoints.
- Ran focused `go test ./cmd/runecode-tui` profiling passes covering shell view and watch-heavy paths.
- Launched PTY-backed TUI sessions against broker local IPC using real `runecode-tui` child PID sampling.
- Re-ran the live measurement after correcting the store-isolation mistake so empty-state and active-state results were not conflated.

## Executed Investigation Results
- Confirmed the real `runecode-tui` child can be sampled directly rather than attributing CPU to the `script` PTY wrapper.
- Confirmed the first live high-CPU sample was against non-empty repo-scoped broker state rather than a truly empty isolated broker store.
- Confirmed the corrected empty-state isolated measurement stayed near `0.5-1.0%` CPU even after the TUI remained open for roughly a minute.
- Confirmed the earlier non-empty-state live sample climbed through `22.81%` and `61.92%` CPU while the shell reported active session state, making it an active or waiting-state sample rather than an empty-idle sample.
- Confirmed focused profiles point more strongly to render, wrap, ANSI, regex, and palette or surface allocation cost than to a single broker request hot spot.

## Planned Automated Checks
- `go test ./cmd/runecode-tui -bench 'BenchmarkShell(View|Watch|BuildPaletteEntries)' -benchmem`
- deterministic PTY-based TUI empty-idle CPU gate
- deterministic PTY-based TUI waiting-state CPU gate
- deterministic broker unary local API latency suite
- deterministic broker watch-family latency suite
- runner boundary and protocol performance suite
- launcher startup and attach-ready performance suite
- audit, protocol, gateway, and project-substrate performance suites

## Verification Notes
- Confirm the change preserves the corrected distinction between empty-state and active-state TUI measurements.
- Confirm the change records the most important methodological correction: socket isolation is not broker-store isolation.
- Confirm the design captures the profile-backed render and allocation hot spots, not just the top-line CPU numbers.
- Confirm performance checks are proposed for all major RuneCode aspects rather than just the TUI.
- Confirm each major subsystem has an explicit threshold policy or regression budget.
- Confirm Linux remains the first authoritative numeric gate while other platforms still execute the same flow families where feasible.
- Confirm the roadmap places this work under `v0.1.0-beta.1`.
- Confirm the change keeps performance verification check-only and CI-safe.

## Close Gate
Use the repository's standard verification flow before closing this change.
