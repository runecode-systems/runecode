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

That file is already part of the project-substrate assurance posture and should remain dedicated to that purpose. This change should instead define one separate reviewed performance-contract artifact family under `runecontext/assurance/performance/` that stores metric identity, fixture identity, environment authority, statistical policy, and threshold declarations for the performance program.

The first artifact family should use:

- `runecontext/assurance/performance/manifest.json` as the reviewed inventory for performance contract files
- per-surface reviewed contract files under `runecontext/assurance/performance/contracts/`
- optional reviewed baseline sample artifacts under `runecontext/assurance/performance/baselines/` only when a metric needs repeated-sample comparison against preserved historical samples
- one trusted repo-local compare/enforce tool under `tools/` that reads these artifacts and check outputs but never rewrites baselines during normal CI

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
- lane authority
- activation state
- baseline source
- comparison method
- practical noise floor
- threshold origin
- notes or review rationale when needed

Each metric contract should also declare:

- `start_event`
- `end_event`
- `clock_source`
- `evidence_source`
- `included_phases`

Those fields make timing boundaries reviewable and prevent implementation from moving a metric to an earlier advisory milestone without changing the reviewed contract.

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

### Lane And Activation States

Each performance metric should declare both a lane authority and an activation state.

Initial lane authorities:

- `required_shared_linux`: required in the Linux PR path on shared hosted Linux because the metric is stable enough with conservative thresholds
- `required_tight_linux`: required for beta closure, but measured in a tighter authoritative Linux environment because shared hosted Linux is too noisy
- `informational_until_stable`: collected in CI or local verification until calibration data proves it is stable enough to become required
- `contract_pending_dependency`: contract and harness may be authored, but the gate cannot become required until the underlying reviewed path exists
- `extended`: non-PR, merge-queue, scheduled, or post-MVP measurement

Initial activation states:

- `defined`: contract exists but no enforcement yet
- `informational`: measurement runs but does not block
- `required`: measurement blocks in its declared lane
- `contract_pending_dependency`: contract exists but depends on another reviewed path before it can run authoritatively

`CHG-025` external-anchor metrics and `CHG-054` truthful-attestation metrics may be defined before those changes fully land, but they must remain `contract_pending_dependency` until the reviewed path exists.

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
| Empty idle CPU | isolated empty broker state, isolated runtime/socket/target alias | sample real `runecode-tui` child CPU for fixed repeated windows after explicit warmup | average `<= 2%`, max sample `<= 4%` | `informational_until_stable`, promote to `required_shared_linux` or `required_tight_linux` after calibration |
| Waiting-state CPU | deterministic waiting session fixture in isolated broker store | sample real `runecode-tui` child CPU for fixed repeated windows after explicit warmup | average `<= 8%`, max sample `<= 12%` | `informational_until_stable`, promote to `required_shared_linux` or `required_tight_linux` after calibration |
| Attach/startup | isolated broker store with no pending work | PTY launch to first settled full frame after broker-owned attachable posture is reached | `<= 500ms` to first settled frame | `required_shared_linux` after timing contract is frozen |
| Key-response latency | quiet route, empty and waiting-state fixtures | fixed repeated trials from key inject to transcript delta proxy | p95 `<= 50ms` empty, p95 `<= 75ms` waiting-state | `required_shared_linux` after sample count is validated |
| Render microbenchmarks | synthetic route surfaces and shell states | `BenchmarkShellViewEmpty`, `BenchmarkShellViewWaitingSession` | fail on `> 15%` regression in `ns/op`, `B/op`, or `allocs/op` from committed Linux baseline once repeated-sample comparison exceeds the reviewed noise floor | `required_shared_linux` |
| Update microbenchmarks | synthetic watch messages and command-surface states | `BenchmarkShellWatchApply`, `BenchmarkBuildPaletteEntries` | fail on `> 15%` regression in `ns/op`, `B/op`, or `allocs/op` once repeated-sample comparison exceeds the reviewed noise floor | `required_shared_linux` |

### Broker Local API, Watch, And Attach Paths

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Unary local API latency | deterministic stores for supported beta fixtures | repeated local trials over `session-list`, `session-get`, `run-list`, `run-get`, `approval-list`, `readiness`, `version-info`, `project-substrate-posture-get` | p95 `<= 150ms` for supported fixture sizes and fail on `> 15%` regression where hybrid budgets are used | `required_shared_linux` |
| Watch-family latency | deterministic stores for supported beta fixtures | repeated local trials over `run-watch`, `approval-watch`, `session-watch`, `session-turn-execution-watch` with `IncludeSnapshot` and `Follow` | p95 `<= 200ms` for supported fixture sizes and fail on `> 15%` regression where hybrid budgets are used | `required_shared_linux` |
| Watch payload growth | same fixtures | response bytes and event counts | fail if payload grows `> 15%` beyond committed baseline per supported fixture bucket | `required_shared_linux` |
| Mutation-path latency | deterministic local stores | `session-execution-trigger`, `continue`, `approval-resolve`, `backend-posture-change` | p95 `<= 200ms` for local control-plane-only paths | `required_shared_linux` |
| Local attach | broker already running with isolated state | attach to ready interactive surface after broker-owned lifecycle posture confirms attachability | `<= 500ms` | `required_shared_linux` after timing contract is frozen |
| Resume after reconnect | persisted session/run state with broker already running | detach and reattach workflow to ready broker-owned session/run truth | `<= 500ms` from attach to ready surface | `required_shared_linux` after timing contract is frozen |

### Runner, Workflow, And Launcher Paths

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Runner boundary check | current repo plus deterministic fixture workspace | `cd runner && npm run boundary-check` | wall time `<= 5s`; fail on `> 15%` regression | `required_shared_linux` |
| Protocol fixture tests | deterministic shared fixture set | `cd runner && node --test scripts/protocol-fixtures.test.js` | wall time `<= 10s`; fail on `> 15%` regression | `required_shared_linux` |
| Representative runner cold start | deterministic minimal workflow fixture | runner startup to first durable checkpoint bound to the active immutable plan identity | `<= 1s` local-overhead budget | `informational_until_stable`, promote after sample stability is proven |
| Supported workflow execution | deterministic MVP workflow fixture | trigger to completed durable broker state on the supported beta slice | threshold derived from committed baseline; fail on `> 15%` regression | `contract_pending_dependency` until the real supported workflow path exists, then promote |
| CHG-050 workflow path | deterministic definitions and process fixtures | validation or canonicalization, trusted compilation, compiled-plan persistence or load, runner startup from immutable `RunPlan` | threshold derived from committed baseline; fail on `> 15%` regression once repeated-sample comparison exceeds the reviewed noise floor | `required_shared_linux` for trusted compilation/load checks; execution startup remains `contract_pending_dependency` until the real path exists |
| MicroVM cold start | deterministic lightweight signed role image with verified-cache miss or required trusted-admission path | trigger to broker-observed ready state | `<= 8s` cold | `informational_until_stable` or `required_tight_linux` after calibration |
| MicroVM warm start | same signed runtime-image fixture with verified local runtime-asset cache hit | trigger to ready | `<= 3s` warm | `informational_until_stable` or `required_tight_linux` after calibration |
| Container cold start | opt-in deterministic signed container-runtime fixture with verified-cache miss or required trusted-admission semantics used for microVM startup checks | trigger to ready | `<= 4s` cold | `informational_until_stable` |
| Container warm start | same signed container-runtime fixture with verified local runtime-asset cache hit | trigger to ready | `<= 2s` warm | `informational_until_stable` |
| Attestation cold path | deterministic runtime startup fixture with full post-handshake verification | launch to persisted post-handshake attestation verification and broker projection | threshold derived from committed baseline; fail on `> 15%` regression | `contract_pending_dependency` until `CHG-054` lands |
| Attestation warm path | same fixture with immutable verification-cache hits | launch to persisted post-handshake attestation verification and broker projection | threshold derived from committed baseline; fail on `> 15%` regression | `contract_pending_dependency` until `CHG-054` lands |

Launcher and attestation checks must preserve the reviewed architecture rather than rewarding unsafe shortcuts:

- cold checks measure trusted admission or verified-cache miss cost when assets are not already locally admitted
- warm checks measure verified-cache hit behavior on the same signed runtime identity
- neither path may reward bypassing signer verification, component-digest checks, attestation verification, replay checks, freshness checks, or launch-deny evidence generation

### Gateway, Dependency, Audit, Protocol, And External Anchor Paths

| Aspect | Fixture | Check | Initial Threshold | CI Lane |
| --- | --- | --- | --- | --- |
| Secret-ingress prepare and submit | stubbed deterministic secret payloads | local broker and secrets overhead only | p95 `<= 300ms` for small payloads | `required_shared_linux` |
| Credential lease issuance | deterministic provider-profile fixture | local issuance overhead | p95 `<= 150ms` | `required_shared_linux` |
| Model-gateway invoke overhead | stubbed provider backend returning deterministic responses | RuneCode-added overhead excluding external network | p95 `<= 100ms` added overhead | `required_shared_linux` |
| Dependency cache miss | deterministic dependency-request fixture and stubbed registry payload source | broker-owned fetch to CAS with no existing cached units | threshold derived from committed baseline; fail on `> 15%` regression in wall time, reviewed bounded-buffer metrics, or peak RSS guardrails beyond reviewed budgets | `required_shared_linux` for bounded-buffer/exact counters; wall/RSS may begin `informational_until_stable` |
| Dependency cache hit | same fixture with cached resolved units already present | broker-owned dependency availability request with no network fetch path taken | threshold derived from committed baseline; fail on `> 15%` regression | `required_shared_linux` |
| Dependency miss coalescing | concurrent identical deterministic dependency requests | wall time, duplicate network work count, and CAS write count | require one effective upstream fill per canonical request identity; fail on duplicate-fill regression | `required_shared_linux` for exact duplicate-fill and CAS-write counts |
| Dependency materialization | deterministic cached dependency manifest and units | broker-mediated offline staging or materialization for workspace use | threshold derived from committed baseline; fail on `> 15%` regression | `required_shared_linux` after fixture calibration |
| Dependency stream-to-CAS posture | large deterministic dependency payload fixture | memory and streaming behavior during cache fill | fail if implementation buffers full payloads in memory beyond reviewed budget, violates reviewed bounded-buffer instrumentation limits, or regresses beyond baseline | `required_shared_linux` for bounded-buffer instrumentation; process RSS starts `informational_until_stable` |
| Audit verification | deterministic ledger fixtures | verify end-to-end locally | threshold derived from committed baseline; fail on `> 15%` regression | `required_shared_linux` |
| Audit finalize verify | deterministic local ledger | finalize plus verify | threshold derived from committed baseline; fail on `> 15%` regression | `required_shared_linux` |
| Protocol schema validation | checked-in protocol schemas and fixtures | schema load and validation suite | `<= 2s` for standard CI fixture set | `required_shared_linux` |
| Fixture-manifest parity | protocol fixtures plus manifest | parity and canonicalization checks | `<= 2s` | `required_shared_linux` |
| External anchor prepare | deterministic sealed audit segment plus stubbed target descriptor | prepare request to durable prepared state | p95 `<= 500ms` local control-plane overhead | `contract_pending_dependency` until `CHG-025` lands |
| External anchor execute-completed | deterministic sealed audit segment plus fast stubbed target | execute request to completed authoritative persistence | threshold derived from committed baseline; fail on `> 15%` regression | `contract_pending_dependency` until `CHG-025` lands |
| External anchor execute-deferred handoff | deterministic sealed audit segment plus intentionally delayed stubbed target | execute request to deferred durable state | p95 `<= 500ms` local control-plane overhead | `contract_pending_dependency` until `CHG-025` lands |
| Deferred completion visibility | same delayed stubbed target | deferred completion to durable completed state plus get/watch visibility | threshold derived from committed baseline; fail on `> 15%` regression | `contract_pending_dependency` until `CHG-025` lands |
| Receipt admission on unchanged seal | already-verified sealed segment plus valid stubbed target proof | authoritative receipt and sidecar admission without full seal replay | threshold derived from committed baseline; fail on `> 15%` regression in wall time or peak RSS | `contract_pending_dependency` until `CHG-025` lands |

External audit anchoring checks must preserve the reviewed architecture rather than rewarding unsafe shortcuts:

- network I/O must stay outside the audit-ledger lock
- deferred execution must remain a first-class lifecycle outcome rather than a hidden test bypass
- unchanged verified seals should use the reviewed incremental receipt-admission path rather than forcing full verifier replay as the only normal path
- checks must not bypass authoritative proof verification, policy binding, or audit evidence persistence to produce a lower number

### Initial Fixture Inventory

The initial gate set should start with a small reviewed fixture inventory rather than a broad ladder.

Recommended first durable slice:

- TUI:
  - `tui.empty.v1`
  - `tui.waiting.v1`
- broker local API:
  - `broker.unary.beta-small.v1`
  - `broker.watch.run.snapshot-follow.v1`
  - `broker.watch.approval.snapshot-follow.v1`
  - `broker.watch.session.snapshot-follow.v1`
  - `broker.watch.turn-execution.snapshot-follow.v1`
- runner and workflow:
  - `workflow.first-party-minimal.v1`
  - `workflow.chg050-compile.v1`
- dependency fetch:
  - `deps.cache-miss.small.v1`
  - `deps.cache-hit.small.v1`
  - `deps.coalesced-miss.small.v1`
- audit:
  - `audit.ledger.standard.v1`
- external anchor:
  - `anchor.fast-complete.stub.v1`
  - `anchor.deferred.stub.v1`
- attestation:
  - `attestation.cold.signed-runtime.v1`
  - `attestation.warm.signed-runtime.v1`

This keeps the first release-defining gate set narrow enough to stay deterministic while still covering the regimes that matter for `v0.1.0-beta.1`.

Fixture IDs are part of metric identity. Future fixture expansion should add new IDs rather than changing these IDs in place unless the fixture semantics intentionally change and the baseline is reviewed as a new contract.

## Statistical And Comparison Policy

### Repeated Microbenchmarks

- Run repeated samples rather than one-off benchmark comparisons.
- Use robust repeated-sample comparison logic suitable for non-normal noisy measurements.
- Require both:
  - the configured regression threshold to be exceeded
  - the change to exceed the reviewed practical noise floor
- Treat summary statistics such as median and confidence intervals as authoritative comparison context, not single best-case runs.

Initial constants:

- required PR comparisons should use at least `10` repeated samples when runtime cost allows
- baseline refresh or threshold recalibration should preferably use at least `20` repeated samples
- fail only when the configured regression threshold and the reviewed practical noise floor are both exceeded
- use a `benchstat`-style comparison workflow or equivalent robust repeated-sample comparison for Go microbenchmarks

### Latency Metrics

- Run a fixed number of repeated trials per fixture.
- Record median and `p95`.
- Gate on the reviewed explicit ceiling, with median retained as diagnostic context.
- Avoid using statistical significance alone as the gate for user-visible latency promises.

Initial constants:

- cheap local latency metrics should target `30` fixed trials so `p95` is meaningful enough for a required gate
- heavier lifecycle metrics may use median plus max ceilings while they are too expensive for a meaningful `p95` sample size
- each latency metric contract should declare whether `p95`, median plus max, or both are authoritative

### CPU And Process-Behavior Metrics

- Use explicit warmup before measurement.
- Use fixed repeated windows.
- Summarize with average or median plus max guardrails.
- Prefer conservative thresholds over false precision in noisy shared environments.

Initial constants:

- each metric must declare warmup duration, observation-window duration, and number of repeated windows before it can become required
- shared hosted Linux CPU metrics should start as `informational_until_stable` unless validation data proves the threshold is stable enough to require there
- max guardrails should catch pathological spikes, but sustained average or median window cost should remain the primary CPU signal

### Exact Metrics

- Treat deterministic counts, duplicate-work counts, payload counts, and similar invariants as exact checks or hard bounds.
- Do not subject exact metrics to inferential comparison logic.

### Threshold Provenance

Every threshold should declare one reviewed `threshold_origin`:

- `product_budget`: an intentional product promise or safety ceiling
- `investigation_baseline`: derived from the corrected investigation data captured by this change
- `first_calibration`: accepted as an initial calibration value after implementation produces repeatable measurements
- `temporary_guardrail`: intentionally provisional and expected to be revisited after more data

Threshold loosening should require explicit review rationale and should not be hidden inside baseline refresh mechanics.

## CI Integration Plan

### Required Linux PR Lane
The required Linux PR lane should include the smallest deterministic `required_shared_linux` checks that still catch the main MVP regressions without pretending high-noise checks are stable on shared hosted runners:

- TUI empty-idle CPU measurement as informational until stability is proven
- TUI waiting-state CPU measurement as informational until stability is proven
- TUI attach/startup gate
- TUI key-response gate
- TUI render and update microbenchmarks
- broker unary local API latency gate
- broker watch-family latency gate
- local attach and resume gates
- protocol and runner deterministic quick checks
- supported workflow execution contracts, with required enforcement only after the real supported workflow path exists
- launcher startup measurements as informational until stability or tighter Linux authority is available
- attestation cold or warm contracts as `contract_pending_dependency` until `CHG-054` lands
- deterministic model-gateway, dependency-fetch, audit, and protocol quick checks
- external-anchor contracts as `contract_pending_dependency` until `CHG-025` lands

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
