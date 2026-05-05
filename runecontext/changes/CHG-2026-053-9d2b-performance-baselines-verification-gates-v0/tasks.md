# Tasks

## Phase 1: Deterministic Fixture Foundation

- [ ] Add deterministic fixture builders for empty and waiting broker stores used by the supported beta path.
- [ ] Add deterministic runner and supported-workflow fixtures that do not require live external dependencies.
- [ ] Add deterministic stubbed provider backends for model-gateway and secrets overhead checks.
- [ ] Add deterministic stubbed external-anchor targets for prepare, execute, deferred, and receipt-admission checks.
- [ ] Define one reviewed performance-contract artifact format for benchmark and latency thresholds, separate from `runecontext/assurance/baseline.yaml`.
- [ ] Store the reviewed performance-contract family under `runecontext/assurance/performance/` with a manifest, per-surface contract files, and optional repeated-sample baseline artifacts where needed.
- [ ] Define one trusted repo-local compare/enforce tool under `tools/` that reads performance contracts and check outputs without rewriting baselines during normal CI.
- [ ] Define the metric taxonomy for exact, absolute-budget, regression-budget, and hybrid-budget checks in the reviewed performance-contract artifacts.
- [ ] Define lane authority and activation states for every metric: `required_shared_linux`, `required_tight_linux`, `informational_until_stable`, `contract_pending_dependency`, and `extended`.
- [ ] Define the initial reviewed MVP fixture inventory per major surface and explicitly defer larger fixture ladders to `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.
- [ ] Assign stable fixture IDs for the initial inventory before collecting baselines.

## Phase 2: TUI Regime Checks

- [ ] Add a real-child CPU sampler for PTY-launched `runecode-tui` suitable for CI.
- [ ] Add an empty-state idle CPU gate using fully isolated broker store, audit ledger, runtime directory, socket, and TUI target alias.
- [ ] Add a waiting-state CPU gate using a deterministic waiting-session fixture.
- [ ] Add attach/startup and key-response latency checks for quiet and waiting-state fixtures.
- [ ] Freeze authoritative timing boundaries for TUI attach and key-response checks, including `start_event`, `end_event`, `clock_source`, `evidence_source`, and `included_phases`.
- [x] Add `go test -bench` coverage for render and update hot paths, including shell view, watch apply, and palette-entry building.

Alpha.7 bootstrap already landed:

- [x] Distinguish waiting activity from actively running work in shell projection and shell chrome so waiting sessions and runs stay visible without reusing the fast running animation loop.

## Phase 3: Broker, Attach, And Resume Checks

- [ ] Add deterministic latency checks for broker unary local API requests used by the supported beta surface.
- [ ] Add deterministic latency and payload-growth checks for `run-watch`, `approval-watch`, `session-watch`, and `session-turn-execution-watch` on the supported beta fixtures.
- [ ] Add control-plane mutation latency checks for execution trigger, continue, approval resolve, and backend posture change paths.
- [ ] Add attach and resume performance checks for the persistent local control-plane lifecycle.
- [ ] Ensure all broker performance checks remain local-only and do not rely on live network services.
- [ ] Freeze authoritative timing boundaries for local attach and resume, including `start_event`, `end_event`, `clock_source`, `evidence_source`, and `included_phases`.

## Phase 4: Runner, Workflow, Launcher, And Attestation Checks

- [ ] Add wall-time and regression checks for runner boundary verification and protocol fixture tests.
- [ ] Add a representative runner cold-start check with a deterministic minimal workflow.
- [ ] Add deterministic checks for the supported MVP workflow execution path.
- [ ] Add deterministic checks for CHG-050 workflow-definition/process-definition validation, canonicalization, and trusted compilation overhead.
- [ ] Add deterministic checks for compiled `RunPlan` persistence/load and runner startup from immutable `RunPlan`.
- [ ] Add deterministic checks for the supported CHG-049 first-party workflow-pack beta slice only.
- [ ] Add cold and warm microVM startup checks, with cold covering verified-cache miss or trusted-admission cost and warm covering verified local runtime-asset cache-hit cost on the same signed runtime identity.
- [ ] Add cold and warm container startup checks for the explicit opt-in backend, with the same verified-cache miss or hit semantics used for microVM startup checks.
- [ ] Add attestation cold-path and warm verification-cache checks for the truthful supported runtime path.
- [ ] Freeze authoritative timing boundaries for launcher and attestation checks, including `start_event`, `end_event`, `clock_source`, `evidence_source`, and `included_phases`.
- [ ] Keep attestation performance contracts in `contract_pending_dependency` until `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` lands.

## Phase 5: Gateway, Dependency, Audit, Protocol, And External Anchor Checks

- [ ] Add deterministic model-gateway invoke-overhead and secret-ingress checks using stubbed provider backends.
- [ ] Add deterministic dependency-fetch cache-miss checks using reviewed typed dependency-request fixtures and stubbed public-registry payload sources.
- [ ] Add deterministic dependency-fetch cache-hit checks for already-cached resolved dependency units.
- [ ] Add miss-coalescing checks so identical concurrent dependency requests do not multiply upstream fetch work.
- [ ] Add broker-mediated offline dependency staging or materialization checks for workspace consumption.
- [ ] Add streaming and memory-budget checks to ensure dependency cache fill stays stream-to-CAS rather than full-payload buffering.
- [ ] Add reviewed bounded-buffer instrumentation for dependency cache fill so stream-to-CAS posture is verified directly in addition to coarse process memory observations.
- [ ] Add audit verification and finalize-verify runtime checks for deterministic ledger fixtures.
- [ ] Add protocol schema and fixture-parity performance checks.
- [ ] Add deterministic external audit anchor prepare checks against stubbed target descriptors and pre-sealed audit segments.
- [ ] Add deterministic external audit anchor execute checks for both fast-completed and deferred-completion paths.
- [ ] Add deferred-completion visibility checks for external audit anchoring through durable get or watch surfaces.
- [ ] Add external anchor receipt-admission checks for unchanged verified seals so the incremental path is measured explicitly.
- [ ] Add checks ensuring external audit anchoring performance does not reward network I/O under audit-ledger lock or bypass authoritative verifier admission.
- [ ] Freeze authoritative timing boundaries for external audit anchoring, including `start_event`, `end_event`, `clock_source`, `evidence_source`, and `included_phases`.
- [ ] Keep external-audit-anchor performance contracts in `contract_pending_dependency` until `CHG-2026-025-5679-external-audit-anchoring-v0` lands.

## Phase 6: CI Integration

- [ ] Add a required Linux PR lane containing the smallest deterministic performance gates with the highest regression value across the MVP beta surface.
- [ ] Limit the initial required shared-Linux PR lane to metrics declared `required_shared_linux` and keep higher-noise metrics informational or pending until their authority is reviewed.
- [ ] Keep performance verification check-only and aligned with `just ci` discipline.
- [ ] Store reviewed threshold declarations in the dedicated performance-contract artifacts rather than auto-generated mutable baselines.
- [ ] Distinguish metrics stable enough for shared hosted Linux required gates from metrics that may later need a tighter authoritative Linux environment.

## Phase 7: Baseline Governance

- [ ] Define the review process for tightening thresholds or accepting deliberate regressions with explicit justification.
- [ ] Document how to refresh baselines safely when major architectural shifts land.
- [ ] Document the reviewed statistical defaults for microbenchmarks, latency metrics, CPU/process-behavior metrics, and exact metrics.
- [ ] Document initial statistical constants for sample counts, warmup windows, p95 eligibility, and repeated-window CPU/process metrics.
- [ ] Document the practical noise-floor policy used alongside repeated-sample regression checks.
- [ ] Document `threshold_origin` for every threshold as `product_budget`, `investigation_baseline`, `first_calibration`, or `temporary_guardrail`.
- [ ] Document which broader performance surfaces are intentionally deferred to `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.

## Acceptance Criteria

- [ ] RuneCode has explicit deterministic performance checks for the supported MVP beta surfaces rather than only anecdotal local measurements.
- [ ] The TUI has separate gates for empty-idle and waiting-state behavior.
- [ ] Broker local API requests and watch families have deterministic latency checks for the supported beta fixtures.
- [ ] Runner startup, the supported workflow path, launcher startup, and the truthful attestation path each have at least one deterministic CI-compatible performance check.
- [ ] Model-gateway, dependency-fetch, audit, protocol, and external audit anchoring paths each have at least one deterministic CI-compatible performance check.
- [ ] Linux PR CI enforces numeric thresholds for the highest-value checks.
- [ ] Reviewed performance-contract artifacts remain separate from project-substrate assurance baseline artifacts.
- [ ] Each required metric has reviewed lane authority, activation state, fixture ID, threshold origin, and timing-boundary metadata.
- [ ] Timing boundaries for attach, workflow, launcher, attestation, dependency, and external-anchor checks terminate on reviewed broker-owned or persisted milestones rather than advisory shortcuts.
- [ ] The first implementation slice uses the reviewed statistical defaults captured by this change and tunes them only through explicit follow-up review.
- [ ] Threshold changes and baseline refreshes require explicit review rather than silent CI mutation.
- [ ] Broader workflow-pack surfaces, git-gateway checks, larger fixture ladders, and tuned macOS or Windows numeric gates are explicitly deferred to `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.
