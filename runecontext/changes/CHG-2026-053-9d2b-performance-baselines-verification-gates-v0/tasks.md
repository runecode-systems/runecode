# Tasks

## Phase 1: Deterministic Fixture Foundation

- [x] Add deterministic fixture builders for empty and waiting broker stores used by the supported beta path.
- [x] Add deterministic runner and supported-workflow fixtures that do not require live external dependencies.
- [x] Add deterministic stubbed provider backends for model-gateway and secrets overhead checks.
- [x] Add deterministic stubbed external-anchor targets for prepare, execute, deferred, and receipt-admission checks.
- [x] Define one reviewed performance-contract artifact format for benchmark and latency thresholds, separate from `runecontext/assurance/baseline.yaml`.
- [x] Store the reviewed performance-contract family under `tools/perfcontracts/` with a manifest, per-surface contract files, reviewed fixture inventory, and optional repeated-sample baseline artifacts where needed.
- [x] Define one trusted repo-local compare/enforce tool under `tools/` that reads performance contracts and check outputs without rewriting baselines during normal CI.
- [x] Define the metric taxonomy for exact, absolute-budget, regression-budget, and hybrid-budget checks in the reviewed performance-contract artifacts.
- [x] Define lane authority and activation states for every metric: `required_shared_linux`, `required_tight_linux`, `informational_until_stable`, `contract_pending_dependency`, and `extended`.
- [x] Define the initial reviewed MVP fixture inventory per major surface and explicitly defer larger fixture ladders to `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.
- [x] Assign stable fixture IDs for the initial inventory before collecting baselines.

## Phase 2: TUI Regime Checks

- [x] Add a real-child CPU sampler for PTY-launched `runecode-tui` suitable for CI.
- [x] Add an empty-state idle CPU gate using fully isolated broker store, audit ledger, runtime directory, socket, and TUI target alias.
- [x] Add a waiting-state CPU gate using a deterministic waiting-session fixture.
- [x] Add attach/startup and key-response latency checks for quiet and waiting-state fixtures.
- [x] Freeze authoritative timing boundaries for TUI attach and key-response checks, including `start_event`, `end_event`, `clock_source`, `evidence_source`, and `included_phases`.
- [x] Add `go test -bench` coverage for render and update hot paths, including shell view, watch apply, and palette-entry building.

Alpha.7 bootstrap already landed:

- [x] Distinguish waiting activity from actively running work in shell projection and shell chrome so waiting sessions and runs stay visible without reusing the fast running animation loop.

## Phase 3: Broker, Attach, And Resume Checks

- [x] Add deterministic latency checks for broker unary local API requests used by the supported beta surface.
- [x] Add deterministic latency and payload-growth checks for `run-watch`, `approval-watch`, `session-watch`, and `session-turn-execution-watch` on the supported beta fixtures.
- [x] Add control-plane mutation latency checks for execution trigger, continue, approval resolve, and backend posture change paths.
- [x] Add attach and resume performance checks for the persistent local control-plane lifecycle.
- [x] Ensure all broker performance checks remain local-only and do not rely on live network services.
- [x] Freeze authoritative timing boundaries for local attach and resume, including `start_event`, `end_event`, `clock_source`, `evidence_source`, and `included_phases`.

## Phase 4: Runner, Workflow, Launcher, And Attestation Checks

- [x] Add wall-time and regression checks for runner boundary verification and protocol fixture tests.
- [x] Add a representative runner cold-start check with a deterministic minimal workflow.
- [x] Add deterministic checks for the supported MVP workflow execution path.
- [x] Add deterministic checks for CHG-050 workflow-definition/process-definition validation, canonicalization, and trusted compilation overhead.
- [x] Add deterministic checks for compiled `RunPlan` persistence/load and runner startup from immutable `RunPlan`.
- [x] Add deterministic checks for the supported CHG-049 first-party workflow-pack beta slice only.
- [x] Add cold and warm microVM startup checks, with cold covering verified-cache miss or trusted-admission cost and warm covering verified local runtime-asset cache-hit cost on the same signed runtime identity.
- [x] Add cold and warm container startup checks for the explicit opt-in backend, with the same verified-cache miss or hit semantics used for microVM startup checks.
- [x] Add attestation cold-path and warm verification-cache checks for the truthful supported runtime path.
- [x] Freeze authoritative timing boundaries for launcher and attestation checks, including `start_event`, `end_event`, `clock_source`, `evidence_source`, and `included_phases`.
- [x] Keep attestation performance contracts in `contract_pending_dependency` until `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` lands.

## Phase 5: Gateway, Dependency, Audit, Protocol, And External Anchor Checks

- [x] Add deterministic model-gateway invoke-overhead and secret-ingress checks using stubbed provider backends.
- [x] Add deterministic dependency-fetch cache-miss checks using reviewed typed dependency-request fixtures and stubbed public-registry payload sources.
- [x] Add deterministic dependency-fetch cache-hit checks for already-cached resolved dependency units.
- [x] Add miss-coalescing checks so identical concurrent dependency requests do not multiply upstream fetch work.
- [x] Add broker-mediated offline dependency staging or materialization checks for workspace consumption.
- [x] Add streaming and memory-budget checks to ensure dependency cache fill stays stream-to-CAS rather than full-payload buffering.
- [x] Add reviewed bounded-buffer instrumentation for dependency cache fill so stream-to-CAS posture is verified directly in addition to coarse process memory observations.
- [x] Add audit verification and finalize-verify runtime checks for deterministic ledger fixtures.
- [x] Add protocol schema and fixture-parity performance checks.
- [x] Add deterministic external audit anchor prepare checks against stubbed target descriptors and pre-sealed audit segments.
- [x] Add deterministic external audit anchor execute checks for both fast-completed and deferred-completion paths.
- [x] Add deferred-completion visibility checks for external audit anchoring through durable get or watch surfaces.
- [x] Add external anchor receipt-admission checks for unchanged verified seals so the incremental path is measured explicitly.
- [x] Add checks ensuring external audit anchoring performance does not reward network I/O under audit-ledger lock or bypass authoritative verifier admission.
- [x] Freeze authoritative timing boundaries for external audit anchoring, including `start_event`, `end_event`, `clock_source`, `evidence_source`, and `included_phases`.
- [x] Keep external-audit-anchor performance contracts in `contract_pending_dependency` until `CHG-2026-025-5679-external-audit-anchoring-v0` lands.

## Phase 6: CI Integration

- [x] Add a required Linux PR lane containing the smallest deterministic performance gates with the highest regression value across the MVP beta surface.
- [x] Limit the initial required shared-Linux PR lane to metrics declared `required_shared_linux` and keep higher-noise metrics informational or pending until their authority is reviewed.
- [x] Keep performance verification check-only and aligned with `just ci` discipline.
- [x] Store reviewed threshold declarations in the dedicated performance-contract artifacts rather than auto-generated mutable baselines.
- [x] Distinguish metrics stable enough for shared hosted Linux required gates from metrics that may later need a tighter authoritative Linux environment.

## Phase 7: Baseline Governance

- [x] Define the review process for tightening thresholds or accepting deliberate regressions with explicit justification.
- [x] Document how to refresh baselines safely when major architectural shifts land.
- [x] Document the reviewed statistical defaults for microbenchmarks, latency metrics, CPU/process-behavior metrics, and exact metrics.
- [x] Document initial statistical constants for sample counts, warmup windows, p95 eligibility, and repeated-window CPU/process metrics.
- [x] Document the practical noise-floor policy used alongside repeated-sample regression checks.
- [x] Document `threshold_origin` for every threshold as `product_budget`, `investigation_baseline`, `first_calibration`, or `temporary_guardrail`.
- [x] Document which broader performance surfaces are intentionally deferred to `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.

## Acceptance Criteria

- [x] RuneCode has explicit deterministic performance checks for the supported MVP beta surfaces rather than only anecdotal local measurements.
- [x] The TUI has separate gates for empty-idle and waiting-state behavior.
- [x] Broker local API requests and watch families have deterministic latency checks for the supported beta fixtures.
- [x] Runner startup, the supported workflow path, launcher startup, and the truthful attestation path each have at least one deterministic CI-compatible performance check.
- [x] Model-gateway, dependency-fetch, audit, protocol, and external audit anchoring paths each have at least one deterministic CI-compatible performance check.
- [x] Linux PR CI enforces numeric thresholds for the highest-value checks.
- [x] Reviewed performance-contract artifacts remain separate from project-substrate assurance baseline artifacts.
- [x] Each required metric has reviewed lane authority, activation state, fixture ID, threshold origin, and timing-boundary metadata.
- [x] Timing boundaries for attach, workflow, launcher, attestation, dependency, and external-anchor checks terminate on reviewed broker-owned or persisted milestones rather than advisory shortcuts.
- [x] The first implementation slice uses the reviewed statistical defaults captured by this change and tunes them only through explicit follow-up review.
- [x] Threshold changes and baseline refreshes require explicit review rather than silent CI mutation.
- [x] Broader workflow-pack surfaces, git-gateway checks, larger fixture ladders, and tuned macOS or Windows numeric gates are explicitly deferred to `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`.
