# Tasks

## Freeze Shared Workflow-Kernel Semantics

- [x] Freeze canonical approval lifecycle semantics:
  - distinguish decision acceptance from approval consumption
  - keep consumption broker-only
  - require consumption to be atomic with trusted application of the exact bound continuation when it occurs
  - keep `consumed`, `denied`, `expired`, `cancelled`, and `superseded` terminal
- [x] Freeze stage-sign-off semantics:
  - add a canonical `StageSummary` contract for sign-off binding
  - define `stage_summary_hash` as the RFC 8785 JCS hash of that contract
  - keep `summary_revision` monotonic but non-authoritative by itself
  - supersede stale sign-off requests when the canonical stage summary changes for the same logical stage scope under the active plan
- [x] Freeze gate-evidence authority semantics:
  - runner reports remain advisory
  - broker validates gate identity, attempt identity, plan binding, and normalized inputs
  - broker materializes the canonical trusted gate-evidence artifact or reference
- [x] Freeze public lifecycle versus partial-blocking semantics:
  - public lifecycle is `blocked` only when no eligible work can progress
  - partial blocking remains coordination and detail state
- [x] Freeze a closed `v0` transition-obligation matrix for required audit evidence before authoritative state advances.

Parallelization: this semantic freeze is foundational; later spec, CI, and traceability work should build on one explicit kernel rather than discovering semantics piecemeal.

## Enhance Protocol And Trusted Runtime Contracts

- [x] Add a canonical `StageSummary` protocol object for stage-sign-off binding and keep `RunStageSummary` as a derived broker read model rather than the trust-root sign-off object.
- [x] Update `ActionPayloadStageSummarySignOff`, approval binding and detail surfaces, and broker stage-sign-off supersession logic to use the canonical `StageSummary` contract and digest.
- [x] Update `protocol/schemas/manifest.json` and any affected protocol fixtures or registries required by new or tightened schema families.
- [x] Freeze and document the compiled effective-policy-context hashing contract that backs `manifest_hash`.
- [x] Align trusted approval storage, broker read-model semantics, and resolution flows with the canonical distinction between decision acceptance and approval consumption.
- [x] Tighten gate-override and canonical gate-evidence contracts so trusted continuation always binds exact gate, attempt, failed-result, and policy-context identities; update schema constraints or explicit trusted validation rules accordingly.
  - Existing protocol/runtime surfaces already satisfied most of this binding story; this change explicitly adopts them as the formal-model baseline and traces them in `traceability.md` and `references.md`.
- [x] Keep read-model summaries and local API convenience responses explicitly derived or operator-facing; do not let them become trust-root objects for the formal model.
- [x] Add a checked-in transition-obligation matrix or adjacent contract artifact that maps authoritative transitions to the required audit facts consumed by the model and verification story.

Parallelization: protocol and trusted-runtime refinements can overlap with spec authoring, but they should land early enough that the model binds the real contracts rather than placeholders.

## Write TLA+ Security-Kernel Spec

- [x] Model a bounded system of:
  - runs
  - plans
  - stage summaries
  - action requests
  - approvals
  - gate attempts
  - gate evidence references
  - broker-authoritative state
  - runner-advisory durable state
- [x] Treat `manifest_hash`, `action_request_hash`, `stage_summary_hash`, relevant artifact digests, and policy-input hashes as opaque deterministic tokens in `v0`.
- [x] Bind stage-sign-off behavior to the canonical `StageSummary` contract rather than to local API read-model summaries.
- [x] Encode canonical finite vocabularies for:
  - approval lifecycle
  - public run lifecycle
  - approval binding kind
  - gate lifecycle and gate-attempt outcomes
  - minimal audit-obligation states
- [x] Prove at least:
  - approval scope isolation
  - single-use approval consumption
  - stage-sign-off supersession
  - broker-wins authority over runner-advisory state
  - correct failed-gate-to-override linkage
  - partial blocking separate from public lifecycle
  - required audit obligations for the closed `v0` transition matrix
- [x] Keep the spec structured so later engines or larger models can extend it without rewriting the kernel module.

Parallelization: can proceed once the semantic freeze above is settled; keep the model rooted in canonical schemas and trusted-service authority boundaries.

## TLC CI Model Checking

- [x] Add checked-in TLC configs with small but meaningful bounds:
  - at least two runs
  - multiple approvals
  - exact-action and stage-sign-off bindings
  - multiple gate attempts
  - success, denial, supersession, expiry, and replay paths
- [x] Add a dedicated `just` recipe for model checking.
- [x] Add the required TLA+ and TLC tooling to the dev shell and CI environment explicitly.
- [x] Run model checking in `just ci` and fail closed on invariant violations.
- [x] Keep the repo clean and deterministic after local or CI model checking.

Parallelization: can be implemented in parallel with spec authoring once the toolchain and CI ownership are agreed.

## Traceability

- [x] Add a maintained mapping between modeled concepts and invariants and:
  - protocol schema families
  - trusted Go modules
  - runner durability modules
  - relevant change IDs and standards
- [x] Keep traceability close enough to the spec and CI output that an invariant failure is actionable without manual archaeology.

Parallelization: can proceed in parallel with spec authoring; it depends on stable module and schema ownership names.

## Acceptance Criteria

- [x] CI fails on workflow-kernel invariant violations.
- [x] The first formal model covers the shared workflow kernel rather than feature-local approximations.
- [x] The change explicitly owns the protocol and trusted-runtime refinements needed to implement the model, rather than leaving them for a separate follow-on change.
- [x] Approval lifecycle, consumption, and supersession semantics are explicitly frozen for later broker, runner, gateway, and TUI work.
- [x] Stage sign-off binds a canonical `StageSummary` contract and hash, with `RunStageSummary` left as a derived read model and `summary_revision` treated as monotonic metadata rather than the trust root by itself.
- [x] Runner-supplied gate evidence remains advisory; canonical gate-evidence binding is broker-owned.
- [x] Public run lifecycle stays on the shared broker vocabulary, and partial blocking remains a detail or coordination concern.
- [x] The `v0` transition-obligation matrix makes the required audit facts for authoritative state advances explicit and modeled.
- [x] The spec and its invariants are traceable to concrete schemas, modules, and standards.
