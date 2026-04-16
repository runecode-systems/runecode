# Design

## Overview
Define and continuously model-check the shared workflow security kernel with TLA+ in CI. The first delivery should not attempt to model the whole product. It should formalize the canonical authority, approval, gate, and lifecycle semantics that later changes must reuse.

## Foundation Goal
The purpose of this change is to freeze the semantics at the narrowest trusted kernel where subtle drift would be most dangerous:
- approval request and decision binding
- approval consumption and supersession
- broker-authoritative versus runner-advisory state
- immutable run-plan and gate-attempt linkage
- partial blocking versus public lifecycle
- minimal audit obligations for authoritative transitions

The formal model should treat existing protocol schemas, trusted Go services, and runner durability rules as the source material. It should not invent a simplified product-local semantics that later code must work around.
This change also owns the minimal contract refinements required when the current protocol or trusted runtime does not yet expose the semantics cleanly enough for formalization.

## Scope Of The First Model
The first TLA+ model should cover a bounded system of:
- runs
- plans
- stage summaries
- action requests
- approvals
- gate attempts
- gate evidence references
- broker-authoritative state
- runner-advisory durable state
- minimal audit obligation facts

For `v0`, the model may treat `manifest_hash`, `action_request_hash`, `stage_summary_hash`, artifact digests, and policy-input hashes as opaque deterministic tokens rather than re-modeling their byte-level construction.

## Key Decisions
- Formal methods focus on the shared workflow security kernel, not arbitrary model reasoning.
- TLC is the authoritative model checker for `v0` CI. The spec should still use a finite-state, Apalache-friendly structure where practical so later expansion does not require a rewrite.
- The first model is approval/run/gate kernel-first, not full-system-first.
- This change owns the small protocol and trusted-runtime refinements needed to make the kernel model implementable; do not defer those gaps to a separate follow-on change.
- The canonical approval lifecycle should distinguish decision acceptance from approval consumption even when some current MVP broker APIs resolve and apply a continuation in one trusted atomic transaction.
- Approval consumption is broker-only and must be atomic with the trusted application of the exact bound continuation when consumption occurs.
- Exact-action approvals bind the canonical `ActionRequest` hash. Stage sign-off approvals bind a canonical stage-summary hash.
- `summary_revision` is monotonic metadata inside the canonical stage-summary contract and helps ordering and UX, but the stage-summary hash remains the trust root.
- Stage-sign-off supersession is keyed to the same logical stage scope under the active plan; older pending or approved sign-off artifacts must not remain consumable after a newer bound stage summary replaces them.
- Runner-supplied gate evidence is advisory input only. The broker validates runner reports against the trusted `RunPlan`, gate identity, attempt identity, and normalized bindings, then materializes the canonical gate-evidence artifact or reference used by trusted read models and audit linkage.
- Public run lifecycle remains on the shared broker vocabulary. A run is publicly `blocked` only when no eligible work can progress; partial blocking remains detail and coordination state.
- The first formal model should encode a small explicit matrix of authoritative transitions that require audit evidence rather than attempting the full audit ledger and receipt semantics in `v0`.
- Traceability is part of the feature, not documentation garnish. Each modeled invariant should map back to concrete schema fields, runtime modules, and change or standard references.

## Implementation-Enabling Contract Refinements Owned By This Change

### Canonical `StageSummary` Contract
- This change should add one canonical protocol object for stage-sign-off binding, referred to here as `StageSummary`.
- Do not overload `RunStageSummary` for this purpose. `RunStageSummary` is a broker read-model summary for local API responses and should remain derived or operator-facing rather than becoming the trust-root sign-off object.
- `ActionPayloadStageSummarySignOff`, approval request and detail binding, broker supersession logic, and the formal model should all bind to the canonical `StageSummary` object or its digest.
- The protocol bundle manifest, related fixtures, and any registries needed for the new or tightened schema family are part of this change.

### Approval Surface Alignment
- Existing approval schemas already carry `approved`, `decided_at`, and `consumed_at` semantics. This change should align trusted storage, broker read models, and resolution flows with that canonical distinction rather than collapsing the semantics to match an MVP shortcut.
- Current convenience APIs may still combine decision acceptance and consumption in one trusted atomic operation, but they should be understood and implemented as an atomic path over the fuller state machine rather than as proof that `approved` is not part of the canonical model.
- If current trusted storage or read models cannot represent accepted-but-not-yet-consumed approvals cleanly, this change should enhance those contracts or make the canonical mapping explicit enough that later async approval flows do not require a semantic reset.

### Effective Policy Context Hash Contract
- This change should freeze the contract behind `manifest_hash` so the formal model and runtime agree on what it means.
- The frozen contract should state which compiled inputs make up the effective policy context and which canonicalization profile produces the digest.
- A protocol-adjacent contract note is acceptable, but the hashing inputs and canonicalization rules must be explicit enough that policy, approval, audit, and the formal model cannot drift.

### Gate-Evidence And Override Contract Tightening
- This change should tighten the trusted binding story for gate overrides and canonical gate evidence.
- Override continuation must bind the exact `gate_id`, `gate_kind`, `gate_version`, `gate_attempt_id`, `overridden_failed_result_ref`, and current policy-context hash.
- Where current schemas leave critical override-link fields optional, this change should either add conditional schema requirements or freeze explicit trusted validation rules captured in the change outputs.
- Canonical gate evidence should retain plan checkpoint and order bindings when available so later retries, overrides, and user-facing inspection surfaces can reuse one stable identity story.

### Audit Transition-Obligation Artifact
- This change should produce one small checked-in transition-obligation matrix or adjacent contract artifact that maps authoritative state transitions to the required audit facts.
- The first TLA+ model may treat those facts abstractly, but the repository should not leave the required transition obligations implicit.

## Semantic Freeze Decisions

### Approval Lifecycle And Consumption
- The canonical foundation this change should freeze is:
  - `pending`
  - `approved`
  - `denied`
  - `expired`
  - `cancelled`
  - `superseded`
  - `consumed`
- `approved` means a valid signed decision has been accepted for the current bound inputs.
- `consumed` means the broker has atomically applied or authorized the exact bound continuation.
- `consumed`, `denied`, `expired`, `cancelled`, and `superseded` are terminal for the approval object.
- One approval may be consumed at most once.
- If an MVP API currently combines decision acceptance and consumption in one trusted operation, that is an implementation shortcut over this state machine, not a reason to collapse the canonical semantics.

### Canonical Stage Summary Contract
- Stage sign-off must bind one canonical `StageSummary` contract, not UI text, `RunStageSummary`, or ad hoc broker-local serialization.
- The canonical stage-summary hash must be computed from RFC 8785 JCS bytes of that contract.
- The contract should carry the smallest semantic set required to explain and authorize sign-off, including:
  - `run_id`
  - `plan_id`
  - `stage_id`
  - `summary_revision`
  - `manifest_hash`
  - stage-scoped capability context
  - requested high-risk capability categories
  - requested gateway or dependency scope changes
  - relevant artifact hashes
- `summary_revision` helps monotonic ordering and supersession tie-breaking, but the stage-summary hash remains the actual trust root.
- `RunStageSummary` may link to or summarize the canonical stage-summary state for operator surfaces, but it must remain a derived read model.

### Gate Evidence Authority
- Untrusted runner reports may propose gate results and evidence details.
- Trusted acceptance of gate outcomes and evidence linkage belongs to the broker.
- The canonical gate-evidence object or reference used by trusted read models, overrides, and audit linkage must be broker-materialized after validation against the active plan and gate bindings.
- Override continuation must link to the exact failed gate result identity and current policy context.

### Public Lifecycle Versus Partial Blocking
- Public run lifecycle stays on:
  - `pending`
  - `starting`
  - `active`
  - `blocked`
  - `recovering`
  - `completed`
  - `failed`
  - `cancelled`
- Partial waits, branch-local blocking, and coordination details remain in run, stage, role, and coordination detail models rather than a second public lifecycle vocabulary.
- A run is publicly `blocked` only when no eligible scope can progress under the active plan and trusted coordination state.

### Minimal Audit Transition Obligations
For `v0`, the model should enumerate and check the minimal authoritative transitions whose audit obligations must remain explicit:
- approval request creation
- approval decision acceptance
- approval consumption
- stage sign-off consumption
- gate result acceptance with canonical evidence linkage
- gate override continuation
- authoritative run terminal transition
- plan supersession or authoritative reconciliation

The first model may represent these obligations as abstract required evidence facts rather than modeling the full audit ledger.

## Protocol And Runtime Surfaces Expected To Change

### Protocol Surfaces
- add a canonical `StageSummary` schema family
- update `ActionPayloadStageSummarySignOff`
- review and tighten as needed:
  - `ApprovalRequest`
  - `ApprovalDecision`
  - `ApprovalSummary`
  - `ApprovalLifecycleDetail`
  - `ApprovalDetail`
  - `GateEvidence`
  - `GateResultReport`
  - `RunnerResultReport`
  - `ActionPayloadGateOverride`
- update `protocol/schemas/manifest.json` and any affected fixtures or registries

### Trusted Runtime Surfaces
- approval storage and canonical lifecycle mapping
- approval resolution and stage-sign-off supersession flow
- policy-context compilation and `manifest_hash` derivation
- gate-evidence validation and canonical materialization
- run-detail lifecycle and coordination projection

### Runner And Tooling Surfaces
- runner durable approval-wait semantics and replay assumptions when approval lifecycle or binding contracts tighten
- protocol fixture or schema parity tests
- dev-shell, `just`, and CI integration for TLA+ and TLC

## Model Structure Recommendation
- Keep one core kernel module that defines the bounded state machines for runs, plans, stage summaries, approvals, action bindings, gate attempts, and the broker or runner authority split.
- Keep a small audit-obligation module or submodule that models required evidence presence for the closed set of authoritative transitions above.
- Keep enumerations and state sets explicit and finite.
- Prefer records, constants, and bounded sets over clever encodings that obscure traceability.

## Initial Invariants To Model
The first formal model should at least prove:
- one run cannot consume another run's approval
- an approval cannot be consumed twice
- a superseded or stale stage sign-off cannot be consumed
- broker-authoritative approval and run truth cannot be replaced by runner-advisory state
- gate override continuation cannot bind to the wrong failed gate result
- partial blocking does not mint a second public lifecycle vocabulary
- authoritative transitions covered by the `v0` matrix cannot advance without their required audit obligations

## CI Model Checking
- Add checked-in TLA+ spec files and deterministic TLC configs.
- Add a dedicated `just` recipe for model checking and include it in `just ci`.
- Add the required tooling to the dev shell and CI environment explicitly; the model-checking path must be deterministic and must leave the repo clean.
- Keep bounds small but meaningful:
  - multiple runs
  - multiple approvals
  - exact-action and stage-sign-off bindings
  - multiple gate attempts
  - success, denial, supersession, expiry, and replay paths

## Traceability
The spec should include a maintained mapping from modeled concepts and invariants to:
- protocol schema families
- trusted Go runtime modules
- runner durability modules
- relevant change IDs and standards

This mapping should be close enough to the spec that a CI failure can be traced to the owning contract without code archaeology.

Maintained artifact location for this change:
- `runecontext/changes/CHG-2026-015-cae6-formal-spec-v0-tla-ci-model-checking/traceability.md`

## Foundation Shortcuts To Avoid
- Do not model UI-local or transport-local semantics as if they were trust roots.
- Do not treat runner-local durable state as public or authoritative truth.
- Do not let `summary_revision` become the trust root instead of the canonical stage-summary hash.
- Do not let runner-supplied gate evidence become authoritative without broker materialization and binding checks.
- Do not collapse decision acceptance and consumption into an unstructured "approval happened" event.
- Do not mint a second public lifecycle enum for partial blocking.
- Do not start with the full audit ledger when a smaller transition-obligation slice is enough to freeze the workflow kernel correctly.

## Main Workstreams
- Shared Workflow-Kernel Semantic Freeze
- Protocol And Trusted-Runtime Contract Refinements
- Canonical Stage Summary Contract
- TLA+ Security-Kernel Model
- Minimal Audit Transition-Obligation Model
- TLC CI Integration
- Traceability Mapping

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
