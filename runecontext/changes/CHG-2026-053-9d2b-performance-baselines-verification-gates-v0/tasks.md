# Tasks

## Phase 1: Deterministic Fixture Foundation

- [ ] Add deterministic fixture builders for empty, waiting, medium, and large broker stores.
- [ ] Add deterministic runner and workflow fixtures that do not require live external dependencies.
- [ ] Add deterministic stubbed provider backends for model-gateway and secrets overhead checks.
- [ ] Add deterministic local bare-remote fixtures for git gateway execution checks.
- [ ] Define one reviewed baseline-artifact format for benchmark and latency thresholds.

## Phase 2: TUI Regime Checks

- [ ] Add a real-child CPU sampler for PTY-launched `runecode-tui` suitable for CI.
- [ ] Add an empty-state idle CPU gate using fully isolated broker store, audit ledger, runtime directory, socket, and TUI target alias.
- [ ] Add a waiting-state CPU gate using a deterministic waiting-session fixture.
- [ ] Add attach/startup and key-response latency checks for quiet and waiting-state fixtures.
- [x] Add `go test -bench` coverage for render and update hot paths, including shell view, watch apply, and palette-entry building.

Alpha.7 bootstrap already landed:

- [x] Distinguish waiting activity from actively running work in shell projection and shell chrome so waiting sessions and runs stay visible without reusing the fast running animation loop.

## Phase 3: Broker Local API And Watch Checks

- [ ] Add deterministic latency checks for broker unary local API requests across 10, 100, and 500 entity fixtures.
- [ ] Add deterministic latency and payload-growth checks for `run-watch`, `approval-watch`, `session-watch`, and `session-turn-execution-watch`.
- [ ] Add control-plane mutation latency checks for execution trigger, continue, approval resolve, and backend posture change paths.
- [ ] Ensure all broker performance checks remain local-only and do not rely on live network services.

## Phase 4: Runner And Workflow Checks

- [ ] Add wall-time and regression checks for runner boundary verification and protocol fixture tests.
- [ ] Add a representative runner cold-start check with a deterministic minimal workflow.
- [ ] Add no-op and small deterministic workflow execution performance checks.
- [ ] Add deterministic checks for CHG-050 workflow-definition/process-definition validation, canonicalization, and trusted compilation overhead.
- [ ] Add deterministic checks for compiled `RunPlan` persistence/load and runner startup from immutable `RunPlan`.
- [ ] Add attach and resume performance checks for persistent local control-plane lifecycle behavior.

## Phase 5: Launcher, Gateway, Audit, And Protocol Checks

- [ ] Add cold and warm microVM startup checks.
- [ ] Add cold and warm container startup checks for the explicit opt-in backend.
- [ ] Add deterministic model-gateway invoke-overhead and secret-ingress checks using stubbed provider backends.
- [ ] Add deterministic dependency-fetch cache-miss checks using reviewed typed dependency-request fixtures and stubbed public-registry payload sources.
- [ ] Add deterministic dependency-fetch cache-hit checks for already-cached resolved dependency units.
- [ ] Add miss-coalescing checks so identical concurrent dependency requests do not multiply upstream fetch work.
- [ ] Add broker-mediated offline dependency staging or materialization checks for workspace consumption.
- [ ] Add streaming and memory-budget checks to ensure dependency cache fill stays stream-to-CAS rather than full-payload buffering.
- [ ] Add audit verification and finalize-verify runtime checks for standard and larger fixture ledgers.
- [ ] Add protocol schema and fixture-parity performance checks.
- [ ] Add git gateway prepare and local execute checks plus project-substrate posture and preview or apply checks.

## Phase 6: CI Integration

- [ ] Add a required Linux PR lane containing the smallest deterministic performance gates with the highest regression value.
- [ ] Add an extended Linux lane for larger fixtures, waiting-state TUI checks, launcher startup, and broader end-to-end measurements.
- [ ] Run the same flow families on macOS and Windows where feasible as smoke or trend collection until platform-specific numeric thresholds are tuned.
- [ ] Keep performance verification check-only and aligned with `just ci` discipline.

## Phase 7: Baseline Governance

- [ ] Check in reviewed threshold declarations rather than auto-generated mutable baselines.
- [ ] Define the review process for tightening thresholds or accepting deliberate regressions with explicit justification.
- [ ] Document how to refresh baselines safely when major architectural shifts land.

## Acceptance Criteria

- [ ] RuneCode has explicit performance checks for all major product aspects, not just the TUI.
- [ ] The TUI has separate gates for empty-idle and waiting-state behavior.
- [ ] Broker local API requests and watch families have deterministic latency checks at multiple fixture sizes.
- [ ] Runner, workflow, launcher, model-gateway, audit, protocol, and git gateway paths each have at least one deterministic CI-compatible performance check.
- [ ] The refined CHG-050 workflow path has explicit checks for validation/canonicalization, trusted compilation, compiled-plan persistence/load, and runner startup from immutable `RunPlan`.
- [ ] Dependency-fetch and offline-cache cold-cache, warm-cache, coalescing, and materialization paths each have at least one deterministic CI-compatible performance check.
- [ ] Linux PR CI enforces numeric thresholds for the highest-value checks.
- [ ] macOS and Windows execute the same performance flow families where feasible, at least as smoke or trend gates.
- [ ] Performance baselines assume one topology-neutral workflow/control-plane architecture across constrained and scaled environments rather than separate architecture paths.
- [ ] Threshold changes and baseline refreshes require explicit review rather than silent CI mutation.
