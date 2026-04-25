# Design

## Overview
This change records the corrected performance investigation findings and turns them into a project-wide performance-verification design.

The design goal is not just to benchmark the TUI. It is to give RuneCode one deterministic, CI-compatible performance program spanning:

- TUI idle, active, attach, and render behavior
- broker local API request and watch paths
- runner and workflow execution paths
- launcher backend startup and attach readiness
- model-gateway and secrets overhead
- audit, protocol, and verification costs
- git gateway and project-substrate flows
- end-to-end attach, resume, and execution behavior

## Investigation Scope And Constraints

### Investigation Goals
The investigation was driven by a user report that the TUI:

- raised apparent system CPU from roughly `1-3%` to roughly `15-20%`
- felt laggy even immediately after launch
- became worse the longer it stayed open

The investigation goals were therefore:

- actually launch and exercise the TUI rather than speculate from code alone
- identify whether the issue was true empty-idle CPU or a specific active-state path
- profile likely render, watch, and allocation hot spots
- collect enough evidence to propose performance checks and thresholds for the whole project

### Constraints During Investigation
- no source changes
- use the real TUI and broker, not a plan-only review
- limited host tooling: `perf`, `pidstat`, and `expect` were not available
- terminal measurements therefore used PTY harnessing, `/proc/<pid>/stat`, captured transcripts, and focused `go test` plus `pprof`

## Measurement Methodology And Corrective Finding

### Initial PTY And Profiling Approach
The investigation used:

- throwaway built binaries for `runecode`, `runecode-broker`, and `runecode-tui`
- a PTY harness via `script`
- direct child PID capture through `/proc/<scriptpid>/task/<scriptpid>/children`
- `/proc/<pid>/stat` CPU delta sampling for the real `runecode-tui` child
- focused `go test ./cmd/runecode-tui` runs with CPU and memory profiles

### Runtime-Direction Permission Gotcha
The broker local IPC runtime directory must be `0700`. The investigation encountered and corrected:

- `local ipc startup failed: broker local runtime directory permissions must be 0700: got 755`

This is an environment-setup requirement, not the core performance problem, but it matters for any future PTY-based verification harness.

### Most Important Measurement Correction
The first live "isolated" TUI run was only socket-isolated, not store-isolated.

The investigation initially used a separate:

- `--runtime-dir`
- `--socket-name`

for broker and TUI, but later confirmed that `runecode-broker serve-local --runtime-dir ...` only changes the local IPC socket location. It does not isolate the broker store or audit ledger unless `--state-root` and `--audit-ledger-root` are also provided.

That meant the first live measurement was still reading repo-scoped broker state and inherited preexisting active or waiting sessions.

The corrected empty-state baseline therefore required all of the following to be isolated together:

- broker `--state-root`
- broker `--audit-ledger-root`
- broker `--runtime-dir`
- broker `--socket-name`
- TUI `--runtime-dir`
- TUI `--socket-name`
- a distinct `RUNECODE_TUI_BROKER_TARGET` alias for local preference isolation

This correction materially changed the interpretation of the results and is part of the durable planning record for future performance work.

## Measured Findings

### Corrected Empty-State Baseline
With a truly isolated broker store, audit ledger, runtime directory, and socket, the real `runecode-tui` child process measured:

- fresh idle CPU: `0.50%`
- mid idle CPU: `1.00%`
- aged idle CPU: `0.67%`
- simple key-to-output timing proxy: `31.4ms`

This supports the conclusion that RuneCode TUI empty-state idle CPU is already near the expected low baseline and does not support the broad claim that the TUI inherently idles at `15-20%` or worse with no active work.

### Non-Empty-State Live Sample
Before the store-isolation correction, the real `runecode-tui` child measured:

- fresh CPU: `0.67%`
- mid CPU: `22.81%`
- aged CPU: `61.92%`
- key-to-output timing proxy: `17.9ms`

The captured transcript showed that the shell had entered active live-activity mode and was reporting an active session:

- `active_session=sess-manual-multiwait`

That sample is still useful, but it must be interpreted as an active or waiting-state sample, not an empty-idle sample.

### Terminal Write-Volume Note
The PTY timing trace for the active-state run showed little output over roughly `73s`, which suggests the CPU cost in that regime is not explained solely by high PTY write throughput. The shell can consume significant CPU through internal render, wrap, measurement, and update work even when terminal output is not continuously flooding the screen.

## Source-Level Findings

### TUI Activity And Polling Model
The TUI currently combines several recurring work sources:

- shell watch polling every `2s`
  - `cmd/runecode-tui/shell_watch_transport.go`
- activity animation tick every `120ms` while activity state is `running`
  - `cmd/runecode-tui/shell_watch_transport.go`
  - `cmd/runecode-tui/shell_update.go`
- mouse cell-motion capture by default
  - `cmd/runecode-tui/shell_model.go`

### Active-State Classification
The current activity projection treats several waiting or incomplete conditions as actively progressing:

- run lifecycle text containing values such as `active`, `run`, `progress`, `queue`, `wait`, or `pending`
- approval status containing `pending`, `requested`, or `wait`
- any session with `HasIncompleteTurn == true`
- session status containing `active`, `run`, `progress`, `wait`, or `queued`

That is semantically reasonable for visibility, but it means long-lived waiting sessions can keep the shell in a continuous animation regime even when little visibly changes.

### Full-Surface Render Cost
The current render path does more work than necessary for an animation-only frame change:

- `activeShellSurface()` calls the route's `ShellSurface()` twice
  - once with a base context to derive layout needs
  - again with resolved regions after layout planning
- overlay-height calculation recomputes surface and layout again through `activeShellSurfaceWithoutOverlayHeight()`
- view rendering then rebuilds the whole workbench frame

That means a small state change such as `activityFrame` advancing can still trigger expensive whole-shell recomputation.

### Watch Fan-Out And Discoverability Refresh
Each shell watch application currently:

- updates the watch reduction and projection
- publishes live activity to every route model
- refreshes the shell discoverability index from watch-derived state
- rebuilds palette entries immediately if the palette is open

The route fan-out is somewhat bounded because only a subset of routes currently consume the live-activity message, but the current model is still broader than "active route only" and is worth gating and profiling.

### Watch Transport Semantics
The current watch transport asks for:

- `IncludeSnapshot: true`
- `Follow: true`

for run, approval, and session watch families.

The broker-side watch builders return batches with explicit snapshot, upsert, and terminal event types derived from current summaries. This is deterministic and correct, but it means each watch poll can re-feed a nontrivial amount of state through reduction, projection, discoverability, and render paths.

## Profile Findings

### Render CPU Hot Spots
Focused shell view profiling identified heavy cumulative CPU cost in:

- `github.com/charmbracelet/x/ansi.stringWidth`
- `github.com/charmbracelet/x/cellbuf.Wrap`
- `github.com/charmbracelet/bubbles/textarea.Model.placeholderView`

### Render Allocation Hot Spots
Focused shell view allocation profiling identified major allocators in:

- `github.com/charmbracelet/x/ansi.(*Parser).SetDataSize`
- `regexp/syntax.(*compiler).inst`
- `github.com/runecode-ai/runecode/cmd/runecode-tui.chatRouteModel.ShellSurface`

### Watch And Update Allocation Hot Spots
Focused watch-heavy profiling identified significant allocation cost in:

- `github.com/runecode-ai/runecode/cmd/runecode-tui.shellModel.buildPaletteEntries`
- `github.com/charmbracelet/x/ansi.(*Parser).SetDataSize`
- `regexp/syntax.(*compiler).inst`

### Recorded Profile Totals
The investigation recorded the following notable profile excerpts:

- shell-view-focused memory profile total: `419.63MB`
  - `regexp/syntax.(*compiler).inst`: `119.71MB`
  - `github.com/charmbracelet/x/ansi.(*Parser).SetDataSize`: `113.63MB`
- watch-heavy memory profile total: `749.40MB`
  - `github.com/charmbracelet/x/ansi.(*Parser).SetDataSize`: `234.21MB`
  - `regexp/syntax.(*compiler).inst`: `205.92MB`
- cumulative watch-heavy allocation in `shellModel.buildPaletteEntries`: `138.17MB`
- cumulative shell-view allocation in `chatRouteModel.ShellSurface`: `94.27MB`

### Corrected Interpretation
The corrected interpretation is narrower and more useful than the initial broad concern:

- empty-state idle is roughly acceptable
- active or waiting-session mode can still be too expensive because a `120ms` repaint loop hits a heavy whole-shell render path
- the project therefore needs separate gates for empty idle, active waiting-state cost, render microbenchmarks, and broker or watch paths

## Best-Practice Guidance Collected During Research
External Go, Bubble Tea, Bubbles, Lip Gloss, and `pprof` guidance converged on a consistent set of themes that match the investigation:

- minimize background ticks, polls, and animation frequency when there is no true visible progress requirement
- use lower animation or FPS ceilings when a state is "waiting" rather than actively changing
- avoid unnecessary mouse-motion capture when click and wheel handling is sufficient
- cache or reuse expensive view fragments instead of rebuilding the full surface on small state changes
- avoid repeated width measurement and wrapping work in hot render loops
- avoid rebuilding regex, parser, and search structures on hot paths when inputs have not meaningfully changed
- use benchmark and profile regression gates in CI rather than relying on one-time local profiling

## Durable Product-Level Conclusions

### Conclusion 1: Empty Idle And Active Waiting Must Be Gated Separately
The most important planning correction is that RuneCode should not have one undifferentiated TUI performance gate.

It needs at least two distinct regimes:

- empty or quiescent local state
- active or waiting session state

### Conclusion 2: Waiting State Is The Higher-Risk User Regime
The investigation suggests the likely user-facing performance pain is not the empty shell. It is the long-lived waiting state where:

- activity semantics keep the shell in `running`
- a `120ms` animation tick remains armed
- whole-shell render work remains expensive

Alpha.7 now partially addresses this specific risk by splitting waiting from running in shell activity semantics, preserving visible waiting cues without keeping the `120ms` running animation armed, and adding focused `cmd/runecode-tui` benchmarks for shell view, watch apply, and palette entry construction. The broader architectural work below remains deferred.

### Conclusion 3: Performance Verification Must Be Cross-Cutting
The TUI findings are the most concrete current example, but the same failure mode can exist elsewhere: regressions remain invisible until a human notices because no deterministic subsystem budgets exist in CI.

## Proposed Performance Verification Architecture

### Governing Principles
- Use deterministic local fixtures, seeded stores, stubbed providers, and local bare remotes rather than live external services.
- Keep performance verification check-only and CI-safe.
- Split thresholds by runtime regime and subsystem.
- Use Linux CI as the first authoritative numeric gate.
- Run the same flows on macOS and Windows where feasible, initially as smoke or trend gates until platform-specific thresholds are tuned.
- Combine absolute thresholds with regression thresholds so the project gets both hard ceilings and drift detection.

### Baseline-Maintenance Policy
- For checks already supported by current evidence, commit explicit absolute thresholds immediately.
- For checks without current measured baselines, bootstrap them with deterministic fixture runs and fail on regression beyond the configured percentage from the committed baseline artifact or benchmark snapshot.
- Tighten thresholds intentionally through review rather than letting CI baselines mutate automatically.

## Performance Check Matrix

### TUI Surface

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Empty idle CPU | isolated empty broker state, isolated runtime/socket/target alias | sample real `runecode-tui` child CPU for 60s | average `<= 2%`, max sample `<= 4%` | required Linux |
| Waiting-state CPU | deterministic waiting session fixture in isolated broker store | sample real `runecode-tui` child CPU for 60s | average `<= 8%`, max sample `<= 12%` | required Linux |
| Attach/startup | isolated broker store with no pending work | PTY launch to first settled full frame | `<= 500ms` to first settled frame | required Linux |
| Key-response latency | quiet route, empty and waiting-state fixtures | key inject to transcript delta proxy | p95 `<= 50ms` empty, p95 `<= 75ms` waiting-state | required Linux |
| Render microbenchmarks | synthetic route surfaces and shell states | `BenchmarkShellViewEmpty`, `BenchmarkShellViewWaitingSession`, `BenchmarkShellViewPaletteOpen` | fail on `> 15%` regression in `ns/op`, `B/op`, or `allocs/op` from committed Linux baseline | required Linux |
| Update microbenchmarks | synthetic watch messages and command-surface states | `BenchmarkShellWatchApply`, `BenchmarkBuildPaletteEntries` | fail on `> 15%` regression in `ns/op`, `B/op`, or `allocs/op` | required Linux |

### Broker Local API And Watch Families

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Unary local API latency | deterministic stores with 10, 100, and 500 entity fixtures | `session-list`, `session-get`, `run-list`, `run-get`, `approval-list`, `readiness`, `version-info`, `project-substrate-posture-get` | p95 `<= 75ms` at 10 items, `<= 150ms` at 100, `<= 300ms` at 500; fail on `> 15%` regression | required Linux |
| Watch-family latency | deterministic stores with 10, 100, and 500 entity fixtures | `run-watch`, `approval-watch`, `session-watch`, `session-turn-execution-watch` with `IncludeSnapshot` and `Follow` | p95 `<= 100ms` at 10 items, `<= 200ms` at 100, `<= 400ms` at 500; fail on `> 15%` regression | required Linux |
| Watch payload growth | same watch fixtures | response bytes and event counts | fail if payload grows `> 15%` beyond committed baseline per fixture bucket | required Linux |
| Mutation-path latency | deterministic local stores | `session-execution-trigger`, `continue`, `approval-resolve`, `backend-posture-change` | p95 `<= 200ms` for local control-plane-only paths | required Linux |

### Runner And Workflow Engine

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Runner boundary check | current repo plus deterministic fixture workspace | `cd runner && npm run boundary-check` | wall time `<= 5s`; fail on `> 15%` regression | required Linux, smoke on macOS/Windows |
| Protocol fixture tests | deterministic shared fixture set | `cd runner && node --test scripts/protocol-fixtures.test.js` | wall time `<= 10s`; fail on `> 15%` regression | required Linux, smoke on macOS/Windows |
| Representative runner cold start | no-op or minimal workflow fixture | runner startup to first durable checkpoint | `<= 1s` local-overhead budget | required Linux |
| Representative workflow execution | deterministic no-op and small-change workflows | trigger to completed durable state | `<= 2s` no-op, `<= 5s` small workflow; fail on `> 15%` regression | extended Linux |

### Control-Plane Attach, Resume, And Session Lifecycle

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Local attach | broker already running with isolated state | attach to ready interactive surface | `<= 500ms` | required Linux |
| Resume after reconnect | persisted session/run state with broker already running | detach and reattach workflow | `<= 500ms` from attach to ready surface | required Linux |
| Session execution orchestration readiness | deterministic waiting and resumed-turn fixtures | time to visible status transition in broker-owned state | `<= 250ms` local control-plane propagation | extended Linux |

### Launcher Backends

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| MicroVM cold start | deterministic lightweight role image | trigger to broker-observed ready state | `<= 8s` cold | extended Linux |
| MicroVM warm start | same fixture with warmed image/toolchain | trigger to ready | `<= 3s` warm | extended Linux |
| Container cold start | opt-in deterministic container backend fixture | trigger to ready | `<= 4s` cold | extended Linux |
| Container warm start | warmed deterministic container fixture | trigger to ready | `<= 2s` warm | extended Linux |

### Model Gateway, Secrets, And Provider Overhead

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Secret-ingress prepare and submit | stubbed deterministic secret payloads | local broker and secrets overhead only | p95 `<= 300ms` for small payloads | extended Linux |
| Credential lease issuance | deterministic provider-profile fixture | local issuance overhead | p95 `<= 150ms` | extended Linux |
| Model-gateway invoke overhead | stubbed provider backend returning deterministic responses | RuneCode-added overhead excluding external network | p95 `<= 100ms` added overhead | extended Linux |

### Audit, Protocol, And Verification Surfaces

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Audit verification | deterministic ledger with 1k and 10k records | verify end-to-end locally | `<= 2s` at 1k, `<= 10s` at 10k | extended Linux |
| Audit finalize verify | deterministic local ledger | finalize plus verify | `<= 3s` at standard CI fixture size | extended Linux |
| Protocol schema validation | checked-in protocol schemas and fixtures | schema load and validation suite | `<= 2s` for standard CI fixture set | required Linux |
| Fixture-manifest parity | protocol fixtures plus manifest | parity and canonicalization checks | `<= 2s` | required Linux |

### Git Gateway And Project Substrate Paths

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Git remote prepare | deterministic local fixture repo | prepare request to response | p95 `<= 500ms` | extended Linux |
| Git execute against local bare remote | local bare remote only, no network | issue execute lease plus execute | `<= 2s` | extended Linux |
| Project substrate posture and preview | deterministic fixture repo | `project-substrate-posture-get`, `adopt`, `init-preview`, `upgrade-preview` | p95 `<= 500ms` for posture and preview flows | extended Linux |
| Project substrate apply | deterministic local fixture repo | `init-apply` or `upgrade-apply` | `<= 2s` for local-only fixture | extended Linux |

## CI Integration Plan

### Required PR Lane
The required Linux PR lane should include the smallest deterministic checks that still catch the main regressions:

- TUI empty-idle CPU gate
- TUI attach/startup gate
- TUI key-response gate
- TUI render and update microbenchmarks
- broker unary local API latency gate
- broker watch-family latency gate
- protocol and runner deterministic quick checks

### Extended Linux Lane
An extended Linux lane, suitable for merge queue or scheduled execution, should include:

- TUI waiting-state CPU gate
- larger 100 and 500 entity broker fixtures
- representative workflow execution checks
- launcher cold and warm backend checks
- model-gateway and secrets overhead checks
- audit and project-substrate heavier checks

### macOS And Windows
Run the same flow families where feasible, but initially use them as:

- smoke gates for correctness of the harness
- trend collection for later threshold tuning
- divergence detection if one platform regresses sharply relative to its own baseline

Linux remains the first authoritative numeric gate until platform-specific noise and baselines are validated.

### Baseline Storage And Review
- Store benchmark baselines and threshold declarations in reviewed repo artifacts.
- Do not auto-rewrite performance baselines inside normal CI runs.
- Threshold changes should require an intentional doc-and-code review path, just like other product contract changes.

## Recommended Optimization Priorities Informed By The Findings
This change mostly records follow-on work rather than implementing it, except for the narrow alpha.7 waiting-state repaint reduction and focused benchmark coverage already landed. The remaining priorities implied by the evidence are:

1. Treat long-lived waiting states differently from visibly progressing states so they do not pay the same `120ms` animation cost.
2. Reduce repeated `ShellSurface()` and layout recomputation in the TUI render path.
3. Narrow live-activity fan-out and discoverability-refresh work where the active route does not need it.
4. Cache or reuse expensive palette, measurement, and wrap work when semantic inputs have not changed.
5. Reevaluate default mouse cell-motion capture if click and wheel handling are sufficient for the intended route behavior.

## Design Risks To Avoid
- Do not create flaky performance gates that depend on live internet, external providers, or shared mutable host state.
- Do not let performance verification introduce writes, mutable lockfiles, or auto-updated baselines into normal CI.
- Do not overfit thresholds to a single developer workstation and then claim they are durable product budgets.
- Do not collapse empty-idle and active-waiting behavior into one TUI metric.
- Do not weaken trust-boundary or audit requirements in the name of performance.
