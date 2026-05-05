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

## Foundation Decisions

### One Architecture Across Constrained And Scaled Environments

This change freezes the same architecture rule already established in the related dependency, workflow, and attestation changes:

- RuneCode must optimize one topology-neutral authority model across Raspberry Pi-class local hardware, ordinary developer workstations, and later vertically or horizontally scaled deployments.
- Performance work must improve the shared broker-owned, audit-preserving, trust-boundary-respecting architecture rather than introducing environment-specific fast paths or alternate authority models.
- No metric or gate may reward implementation shortcuts that bypass policy, audit, replay protection, attestation order, broker-owned lifecycle truth, or other reviewed control-plane responsibilities.

### Separate Performance Contract Artifacts

Performance baselines for this change must not be stored in `runecontext/assurance/baseline.yaml`.

That file is already part of the project-substrate assurance posture and should remain dedicated to that purpose. This change should instead define one separate reviewed performance-contract artifact family that stores metric identity, fixture identity, environment authority, statistical policy, and threshold declarations for the performance program.

The first reviewed artifact family should be explicit enough to capture at least:

- `metric_id`
- subsystem or surface identity
- runtime regime identity
- fixture identity
- measurement kind and unit
- authoritative environment
- sampling policy
- budget class
- explicit threshold or regression allowance
- notes or review rationale when needed

### Metric Taxonomy

The first gate set should freeze one metric taxonomy so each check uses the right contract model instead of one generic performance bucket:

- exact checks:
  - deterministic event counts
  - duplicate-work counts
  - CAS write counts
  - other invariant counts that should not vary across runs
- absolute budgets:
  - user-visible attach and startup ceilings
  - key-response ceilings
  - CPU and similar operator-visible ceilings where the product promise is explicit
- regression budgets:
  - repeated microbenchmarks
  - stable allocation-heavy hot paths
  - stable deterministic verification suites where historical regression is the main risk
- hybrid budgets:
  - paths that need both a reviewed absolute product ceiling and a relative regression budget against a checked-in baseline

### Statistical Defaults

The first implementation slice should start with these statistical defaults and tune them only after implementation and validation data shows a concrete need:

- repeated microbenchmarks:
  - use repeated samples rather than one-run comparisons
  - use robust comparison appropriate for noisy non-normal benchmark data
  - require a practical noise-floor threshold in addition to statistical significance so tiny but detectable changes do not cause gate churn
- latency metrics:
  - run a fixed number of repeated trials
  - record median and `p95`
  - gate on explicit reviewed ceilings, with median retained as supporting diagnostic context
- CPU and process-behavior metrics:
  - use fixed observation windows after explicit warmup
  - summarize repeated runs with average or median and a max guardrail
  - avoid pretending that high-noise metrics deserve more inferential precision than the environment can support
- exact metrics:
  - compare as exact values or hard bounds rather than inferential tests

For Go microbenchmarks and similar repeated local measurements, the initial implementation may use a `benchstat`-style comparison workflow or equivalent robust repeated-sample comparison logic, but the durable product rule is the metric-class policy above rather than a tool-specific implementation detail.

### Timing Boundary Rule

Performance timing boundaries must terminate on reviewed authoritative milestones whenever they exist.

That means:

- prefer broker-owned typed lifecycle posture, persisted evidence, persisted verification outputs, or durable broker-projected state over launcher-local, client-local, or transcript-scrape heuristics
- where operator experience and authoritative completion are both important, the design may capture both as separate metrics or sub-metrics, but it must not silently substitute an earlier advisory milestone for the authoritative one

This rule is especially important for:

- attach and resume readiness
- runner startup from immutable `RunPlan`
- signed runtime startup and attach-ready behavior
- truthful post-handshake attestation verification
- external anchor prepare, execute, deferred handoff, and completion visibility

### Fixture Scope Rule

The MVP gate set should start with one small reviewed fixture inventory per major surface rather than broad ladder coverage.

The first durable slice should prefer one golden deterministic fixture per major surface, with additional buckets only where a distinct runtime regime or scalability posture is already product-relevant. Larger fixture ladders, heavier extended lanes, and broader scale confidence work remain explicit post-MVP expansion work under `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.

### Linux Environment Authority

Linux remains the first authoritative numeric-gate environment, but this change should be honest about measurement noise.

- Shared hosted Linux CI is acceptable for the initial required-gate slice where thresholds are conservative enough to remain deterministic.
- The design should allow selected higher-noise metrics to be promoted later to a tighter authoritative Linux environment without changing metric identity or product architecture.
- The performance program must not require a second product architecture merely because measurement infrastructure differs.

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

The first gate set should also preserve the distinction between:

- user-visible experience regimes
- repeated microbenchmark hot paths
- process-behavior resource regimes
- exact-count or invariant-preservation regimes

## Performance Check Matrix

### TUI Surface

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Empty idle CPU | isolated empty broker state, isolated runtime/socket/target alias | sample real `runecode-tui` child CPU for fixed repeated windows after explicit warmup | average `<= 2%`, max sample `<= 4%` | required Linux |
| Waiting-state CPU | deterministic waiting session fixture in isolated broker store | sample real `runecode-tui` child CPU for fixed repeated windows after explicit warmup | average `<= 8%`, max sample `<= 12%` | required Linux |
| Attach/startup | isolated broker store with no pending work | PTY launch to first settled full frame after broker-owned attachable posture is reached | `<= 500ms` to first settled frame | required Linux |
| Key-response latency | quiet route, empty and waiting-state fixtures | fixed repeated trials from key inject to transcript delta proxy | p95 `<= 50ms` empty, p95 `<= 75ms` waiting-state | required Linux |
| Render microbenchmarks | synthetic route surfaces and shell states | `BenchmarkShellViewEmpty`, `BenchmarkShellViewWaitingSession` | fail on `> 15%` regression in `ns/op`, `B/op`, or `allocs/op` from committed Linux baseline once repeated-sample comparison exceeds the reviewed noise floor | required Linux |
| Update microbenchmarks | synthetic watch messages and command-surface states | `BenchmarkShellWatchApply`, `BenchmarkBuildPaletteEntries` | fail on `> 15%` regression in `ns/op`, `B/op`, or `allocs/op` once repeated-sample comparison exceeds the reviewed noise floor | required Linux |

### Broker Local API, Watch, And Attach Paths

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Unary local API latency | deterministic stores for supported beta fixtures | repeated local trials over `session-list`, `session-get`, `run-list`, `run-get`, `approval-list`, `readiness`, `version-info`, `project-substrate-posture-get` | p95 `<= 150ms` for supported fixture sizes and fail on `> 15%` regression where hybrid budgets are used | required Linux |
| Watch-family latency | deterministic stores for supported beta fixtures | repeated local trials over `run-watch`, `approval-watch`, `session-watch`, `session-turn-execution-watch` with `IncludeSnapshot` and `Follow` | p95 `<= 200ms` for supported fixture sizes and fail on `> 15%` regression where hybrid budgets are used | required Linux |
| Watch payload growth | same fixtures | response bytes and event counts | fail if payload grows `> 15%` beyond committed baseline per supported fixture bucket | required Linux |
| Mutation-path latency | deterministic local stores | `session-execution-trigger`, `continue`, `approval-resolve`, `backend-posture-change` | p95 `<= 200ms` for local control-plane-only paths | required Linux |
| Local attach | broker already running with isolated state | attach to ready interactive surface after broker-owned lifecycle posture confirms attachability | `<= 500ms` | required Linux |
| Resume after reconnect | persisted session/run state with broker already running | detach and reattach workflow to ready broker-owned session/run truth | `<= 500ms` from attach to ready surface | required Linux |

### Runner, Workflow, And Launcher Paths

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Runner boundary check | current repo plus deterministic fixture workspace | `cd runner && npm run boundary-check` | wall time `<= 5s`; fail on `> 15%` regression | required Linux |
| Protocol fixture tests | deterministic shared fixture set | `cd runner && node --test scripts/protocol-fixtures.test.js` | wall time `<= 10s`; fail on `> 15%` regression | required Linux |
| Representative runner cold start | deterministic minimal workflow fixture | runner startup to first durable checkpoint bound to the active immutable plan identity | `<= 1s` local-overhead budget | required Linux |
| Supported workflow execution | deterministic MVP workflow fixture | trigger to completed durable broker state on the supported beta slice | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| CHG-050 workflow path | deterministic definitions and process fixtures | validation or canonicalization, trusted compilation, compiled-plan persistence or load, runner startup from immutable `RunPlan` | threshold derived from committed baseline; fail on `> 15%` regression once repeated-sample comparison exceeds the reviewed noise floor | required Linux |
| MicroVM cold start | deterministic lightweight signed role image with verified-cache miss or required trusted-admission path | trigger to broker-observed ready state | `<= 8s` cold | required Linux |
| MicroVM warm start | same signed runtime-image fixture with verified local runtime-asset cache hit | trigger to ready | `<= 3s` warm | required Linux |
| Container cold start | opt-in deterministic signed container-runtime fixture with verified-cache miss or required trusted-admission path | trigger to ready | `<= 4s` cold | required Linux |
| Container warm start | same signed container-runtime fixture with verified local runtime-asset cache hit | trigger to ready | `<= 2s` warm | required Linux |
| Attestation cold path | deterministic runtime startup fixture with full post-handshake verification | launch to persisted post-handshake attestation verification and broker projection | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Attestation warm path | same fixture with immutable verification-cache hits | launch to persisted post-handshake attestation verification and broker projection | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |

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
| Dependency cache miss | deterministic dependency-request fixture and stubbed registry payload source | broker-owned fetch to CAS with no existing cached units | threshold derived from committed baseline; fail on `> 15%` regression in wall time, reviewed bounded-buffer metrics, or peak RSS guardrails beyond reviewed budgets | required Linux |
| Dependency cache hit | same fixture with cached resolved units already present | broker-owned dependency availability request with no network fetch path taken | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Dependency miss coalescing | concurrent identical deterministic dependency requests | wall time, duplicate network work count, and CAS write count | require one effective upstream fill per canonical request identity; fail on duplicate-fill regression | required Linux |
| Dependency materialization | deterministic cached dependency manifest and units | broker-mediated offline staging or materialization for workspace use | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Dependency stream-to-CAS posture | large deterministic dependency payload fixture | memory and streaming behavior during cache fill | fail if implementation buffers full payloads in memory beyond reviewed budget, violates reviewed bounded-buffer instrumentation limits, or regresses beyond baseline | required Linux |
| Audit verification | deterministic ledger fixtures | verify end-to-end locally | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Audit finalize verify | deterministic local ledger | finalize plus verify | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Protocol schema validation | checked-in protocol schemas and fixtures | schema load and validation suite | `<= 2s` for standard CI fixture set | required Linux |
| Fixture-manifest parity | protocol fixtures plus manifest | parity and canonicalization checks | `<= 2s` | required Linux |
| External anchor prepare | deterministic sealed audit segment plus stubbed target descriptor | prepare request to durable prepared state | p95 `<= 500ms` local control-plane overhead | required Linux |
| External anchor execute-completed | deterministic sealed audit segment plus fast stubbed target | execute request to completed authoritative persistence | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| External anchor execute-deferred handoff | deterministic sealed audit segment plus intentionally delayed stubbed target | execute request to deferred durable state | p95 `<= 500ms` local control-plane overhead | required Linux |
| Deferred completion visibility | same delayed stubbed target | deferred completion to durable completed state plus get/watch visibility | threshold derived from committed baseline; fail on `> 15%` regression | required Linux |
| Receipt admission on unchanged seal | already-verified sealed segment plus valid stubbed target proof | authoritative receipt and sidecar admission without full seal replay | threshold derived from committed baseline; fail on `> 15%` regression in wall time or peak RSS | required Linux |

External audit anchoring checks must preserve the reviewed architecture rather than rewarding unsafe shortcuts:

- network I/O must stay outside the audit-ledger lock
- deferred execution must remain a first-class lifecycle outcome rather than a hidden test bypass
- unchanged verified seals should use the reviewed incremental receipt-admission path rather than forcing full verifier replay as the only normal path
- checks must not bypass authoritative proof verification, policy binding, or audit evidence persistence to produce a lower number

### Initial Fixture Inventory

The initial gate set should start with a small reviewed fixture inventory rather than a broad ladder.

Recommended first durable slice:

- TUI:
  - empty fixture
  - waiting fixture
- broker local API:
  - one supported fixture size for unary reads
  - one supported snapshot-plus-follow fixture for each watched family
- runner and workflow:
  - one minimal supported workflow fixture
  - one canonical CHG-050 validation or compilation fixture
- dependency fetch:
  - one cache miss fixture
  - one cache hit fixture
  - one coalesced identical-miss fixture
- audit:
  - one standard local ledger fixture
- external anchor:
  - one fast-complete stubbed target fixture
  - one deferred-completion stubbed target fixture
- attestation:
  - one cold fixture
  - one warm verification-cache fixture on the same signed runtime identity

This keeps the first release-defining gate set narrow enough to stay deterministic while still covering the regimes that matter for `v0.1.0-beta.1`.

## Statistical And Comparison Policy

### Repeated Microbenchmarks

- Run repeated samples rather than one-off benchmark comparisons.
- Use robust repeated-sample comparison logic suitable for non-normal noisy measurements.
- Require both:
  - the configured regression threshold to be exceeded
  - the change to exceed the reviewed practical noise floor
- Treat summary statistics such as median and confidence intervals as authoritative comparison context, not single best-case runs.

### Latency Metrics

- Run a fixed number of repeated trials per fixture.
- Record median and `p95`.
- Gate on the reviewed explicit ceiling, with median retained as diagnostic context.
- Avoid using statistical significance alone as the gate for user-visible latency promises.

### CPU And Process-Behavior Metrics

- Use explicit warmup before measurement.
- Use fixed repeated windows.
- Summarize with average or median plus max guardrails.
- Prefer conservative thresholds over false precision in noisy shared environments.

### Exact Metrics

- Treat deterministic counts, duplicate-work counts, payload counts, and similar invariants as exact checks or hard bounds.
- Do not subject exact metrics to inferential comparison logic.

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

The first required lane should prefer metrics that are already stable enough on shared hosted Linux. The design may later promote selected higher-noise gates to a tighter authoritative Linux environment, but the initial gate set should not depend on that tighter environment existing on day one.

### Baseline Storage And Review
- Store benchmark baselines and threshold declarations in reviewed performance-contract artifacts separate from `runecontext/assurance/baseline.yaml`.
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
- Do not terminate metrics at advisory client-local or launcher-local milestones when authoritative persisted or broker-owned milestones exist downstream in the reviewed product contract.
- Do not use one universal statistics rule for every metric class when the metric semantics clearly differ.
- Do not weaken trust-boundary or audit requirements in the name of performance.
