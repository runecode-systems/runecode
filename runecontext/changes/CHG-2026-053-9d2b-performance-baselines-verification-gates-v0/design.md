# Design

## Overview
This change records the corrected performance investigation findings and turns them into RuneCode's first MVP-grade performance-verification design.

The design goal is not to benchmark every current or future product surface. It is to give RuneCode one deterministic, CI-compatible performance program for the supported `v0.1.0-beta.1` surface spanning:

- TUI idle, waiting, attach, and render behavior
- broker local API request and watch paths
- runner startup and the supported MVP workflow execution path
- launcher backend startup and attach readiness
- required runtime attestation verification and attestation verification-cache behavior
- model-gateway and secrets overhead
- dependency-fetch and offline-cache overhead
- audit, protocol, and verification costs
- external audit anchoring prepare, execute, deferred completion, and receipt-admission costs
- end-to-end attach, resume, and execution behavior on Linux

The broader performance expansion remains a separate post-MVP lane in `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.

## Investigation Scope And Constraints

### Investigation Goals
The investigation was driven by a user report that the TUI:

- raised apparent system CPU from roughly `1-3%` to roughly `15-20%`
- felt laggy even immediately after launch
- became worse the longer it stayed open

The investigation goals were therefore:

- actually launch and exercise the TUI rather than speculate from code alone
- identify whether the issue was true empty-idle CPU or a specific waiting-state path
- profile likely render, watch, and allocation hot spots
- collect enough evidence to propose deterministic MVP beta checks

### Constraints During Investigation
- no source changes during the original investigation
- use the real TUI and broker, not a plan-only review
- limited host tooling: `perf`, `pidstat`, and `expect` were not available
- terminal measurements therefore used PTY harnessing, `/proc/<pid>/stat`, captured transcripts, and focused `go test` plus `pprof`

## Measurement Methodology And Corrective Finding

### Runtime-Direction Permission Gotcha
The broker local IPC runtime directory must be `0700`. The investigation encountered and corrected:

- `local ipc startup failed: broker local runtime directory permissions must be 0700: got 755`

This is an environment-setup requirement, not the core performance problem, but it matters for future PTY-based verification harnesses.

### Most Important Measurement Correction
The first live "isolated" TUI run was only socket-isolated, not store-isolated.

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

This supports the conclusion that RuneCode TUI empty-state idle CPU is already near the expected low baseline.

### Waiting-State Risk
Before the store-isolation correction, a non-empty-state live sample climbed through `22.81%` and `61.92%` CPU while the shell reported active session state. After the alpha.7 waiting-state fix, the isolated waiting-state rerun measured:

- `0.00%` fresh CPU
- `1.00%` mid CPU
- `1.00%` aged CPU

The strongest current evidence is therefore:

- empty-state idle is roughly acceptable
- waiting state was the higher-risk user regime
- the alpha.7 fix materially improved that specific repaint regression

## MVP Performance Regimes
The MVP gate set should distinguish at least these regimes:

- empty or quiescent local state
- waiting-session local state
- attach and resume latency
- render and update hot paths
- broker request and watch latency
- supported workflow startup and execution
- launcher startup and attach-ready behavior
- attestation cold and warm verification cost
- model-gateway, dependency-fetch, audit, protocol, and external-anchor overhead

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

### Broker Local API, Watch, And Attach Paths

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Unary local API latency | deterministic stores for supported beta fixtures | `session-list`, `session-get`, `run-list`, `run-get`, `approval-list`, `readiness`, `version-info`, `project-substrate-posture-get` | p95 `<= 150ms` for supported fixture sizes; fail on `> 15%` regression | required Linux |
| Watch-family latency | deterministic stores for supported beta fixtures | `run-watch`, `approval-watch`, `session-watch`, `session-turn-execution-watch` with `IncludeSnapshot` and `Follow` | p95 `<= 200ms` for supported fixture sizes; fail on `> 15%` regression | required Linux |
| Watch payload growth | same fixtures | response bytes and event counts | fail if payload grows `> 15%` beyond committed baseline per supported fixture bucket | required Linux |
| Mutation-path latency | deterministic local stores | `session-execution-trigger`, `continue`, `approval-resolve`, `backend-posture-change` | p95 `<= 200ms` for local control-plane-only paths | required Linux |
| Local attach | broker already running with isolated state | attach to ready interactive surface | `<= 500ms` | required Linux |
| Resume after reconnect | persisted session/run state with broker already running | detach and reattach workflow | `<= 500ms` from attach to ready surface | required Linux |

### Runner, Workflow, And Launcher Paths

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Runner boundary check | current repo plus deterministic fixture workspace | `cd runner && npm run boundary-check` | wall time `<= 5s`; fail on `> 15%` regression | required Linux |
| Protocol fixture tests | deterministic shared fixture set | `cd runner && node --test scripts/protocol-fixtures.test.js` | wall time `<= 10s`; fail on `> 15%` regression | required Linux |
| Representative runner cold start | deterministic minimal workflow fixture | runner startup to first durable checkpoint | `<= 1s` local-overhead budget | required Linux |
| Supported workflow execution | deterministic MVP workflow fixture | trigger to completed durable state | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| CHG-050 workflow path | deterministic definitions and process fixtures | validation or canonicalization, trusted compilation, compiled-plan persistence or load, runner startup from immutable `RunPlan` | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| MicroVM cold start | deterministic lightweight signed role image with verified-cache miss or required trusted-admission path | trigger to broker-observed ready state | `<= 8s` cold | required Linux |
| MicroVM warm start | same signed runtime-image fixture with verified local runtime-asset cache hit | trigger to ready | `<= 3s` warm | required Linux |
| Container cold start | opt-in deterministic signed container-runtime fixture with verified-cache miss or required trusted-admission path | trigger to ready | `<= 4s` cold | required Linux |
| Container warm start | same signed container-runtime fixture with verified local runtime-asset cache hit | trigger to ready | `<= 2s` warm | required Linux |
| Attestation cold path | deterministic runtime startup fixture with full post-handshake verification | launch to persisted attestation projection | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Attestation warm path | same fixture with immutable verification-cache hits | launch to persisted attestation projection | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |

Launcher and attestation checks must preserve the reviewed architecture rather than rewarding unsafe shortcuts:

- cold checks measure trusted admission or verified-cache miss cost when assets are not already locally admitted
- warm checks measure verified-cache hit behavior on the same signed runtime identity
- neither path may reward bypassing signer verification, component-digest checks, attestation verification, replay checks, freshness checks, or launch-deny evidence generation

### Gateway, Dependency, Audit, Protocol, And External Anchor Paths

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Secret-ingress prepare and submit | stubbed deterministic secret payloads | local broker and secrets overhead only | p95 `<= 300ms` for small payloads | required Linux |
| Credential lease issuance | deterministic provider-profile fixture | local issuance overhead | p95 `<= 150ms` | required Linux |
| Model-gateway invoke overhead | stubbed provider backend returning deterministic responses | RuneCode-added overhead excluding external network | p95 `<= 100ms` added overhead | required Linux |
| Dependency cache miss | deterministic dependency-request fixture and stubbed registry payload source | broker-owned fetch to CAS with no existing cached units | threshold derived from committed baseline; fail on `> 15%` regression in wall time, peak RSS, or bytes buffered beyond reviewed budget | required Linux |
| Dependency cache hit | same fixture with cached resolved units already present | broker-owned dependency availability request with no network fetch path taken | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Dependency miss coalescing | concurrent identical deterministic dependency requests | wall time, duplicate network work count, and CAS write count | require one effective upstream fill per canonical request identity; fail on duplicate-fill regression | required Linux |
| Dependency materialization | deterministic cached dependency manifest and units | broker-mediated offline staging or materialization for workspace use | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Dependency stream-to-CAS posture | large deterministic dependency payload fixture | memory and streaming behavior during cache fill | fail if implementation buffers full payloads in memory beyond reviewed budget or regresses beyond baseline | required Linux |
| Audit verification | deterministic ledger fixtures | verify end-to-end locally | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Audit finalize verify | deterministic local ledger | finalize plus verify | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Protocol schema validation | checked-in protocol schemas and fixtures | schema load and validation suite | `<= 2s` for standard CI fixture set | required Linux |
| Fixture-manifest parity | protocol fixtures plus manifest | parity and canonicalization checks | `<= 2s` | required Linux |
| External anchor prepare | deterministic sealed audit segment plus stubbed target descriptor | prepare request to durable prepared state | p95 `<= 500ms` local control-plane overhead | required Linux |
| External anchor execute-completed | deterministic sealed audit segment plus fast stubbed target | execute request to completed authoritative persistence | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| External anchor execute-deferred handoff | deterministic sealed audit segment plus intentionally delayed stubbed target | execute request to deferred durable state | p95 `<= 500ms` local control-plane overhead | required Linux |
| Deferred completion visibility | same delayed stubbed target | deferred completion to durable completed state plus get or watch visibility | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Receipt admission on unchanged seal | already-verified sealed segment plus valid stubbed target proof | authoritative receipt and sidecar admission without full seal replay | threshold derived from committed baseline; fail on `> 15%` regression in wall time or peak RSS | required Linux |

External audit anchoring checks must preserve the reviewed architecture rather than rewarding unsafe shortcuts:

- network I/O must stay outside the audit-ledger lock
- deferred execution must remain a first-class lifecycle outcome rather than a hidden test bypass
- unchanged verified seals should use the reviewed incremental receipt-admission path rather than forcing full verifier replay as the only normal path
- checks must not bypass authoritative proof verification, policy binding, or audit evidence persistence to produce a lower number

## CI Integration Plan

### Required Linux PR Lane
The required Linux PR lane should include the smallest deterministic checks that still catch the main MVP regressions:

- TUI empty-idle CPU gate
- TUI waiting-state CPU gate
- TUI attach/startup gate
- TUI key-response gate
- TUI render and update microbenchmarks
- broker unary local API latency gate
- broker watch-family latency gate
- local attach and resume gates
- protocol and runner deterministic quick checks
- supported workflow execution gate
- launcher startup and attestation cold or warm quick checks
- deterministic model-gateway, dependency-fetch, audit, and external-anchor quick checks

### Baseline Storage And Review
- Store benchmark baselines and threshold declarations in reviewed repo artifacts.
- Do not auto-rewrite performance baselines inside normal CI runs.
- Threshold changes should require an intentional doc-and-code review path, just like other product contract changes.

## Explicit Deferrals
The following belong to the post-MVP expansion lane, not this MVP gate set:

- broader CHG-049 workflow-pack surfaces beyond the supported beta workflow slice
- git-gateway and broader project-substrate performance suites that are not part of the beta hard gate
- larger broker-fixture ladders and heavier extended-lane measurements beyond the first release-defining fixtures
- tuned macOS and Windows numeric gates and wider cross-platform parity work

Those remain tracked in `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.

## Design Risks To Avoid
- Do not create flaky performance gates that depend on live internet, external providers, or shared mutable host state.
- Do not let performance verification introduce writes, mutable lockfiles, or auto-updated baselines into normal CI.
- Do not overfit thresholds to a single developer workstation and then claim they are durable product budgets.
- Do not collapse empty-idle and waiting-state behavior into one TUI metric.
- Do not weaken trust-boundary or audit requirements in the name of performance.
