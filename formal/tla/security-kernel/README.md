# Security Kernel TLA+ Spec (CHG-2026-015)

This directory contains the checked-in TLA+ artifacts for the bounded workflow security kernel.

## Files

- `SecurityKernelV0.tla` — bounded kernel model of runs, plans, stage summaries, action requests, approvals, gate attempts/evidence, broker-vs-runner authority, execution-scope waits, dependency-aware partial blocking, project-substrate binding/drift, and minimal audit obligations.
- `SecurityKernelV0.core.cfg` — deterministic TLC model focused on core approval/stage/gate/lifecycle invariants.
- `SecurityKernelV0.replay.cfg` — deterministic TLC model that keeps replay/reconciliation paths in scope with slightly broader bounds.
- `SecurityKernelV0_MC.cfg` — compatibility config kept alongside the dedicated model configs.

## Scope Notes

- Opaque deterministic tokens are modeled as bounded sets (`HashTokens`, `ArtifactDigests`, `PolicyHashes`, `PolicyInputHashes`, `EvidenceRefs`) instead of byte-level hashing.
- Stage sign-off semantics are bound to canonical `StageSummary` identity via `stageSummaryHash` (not `RunStageSummary`).
- Runner durable state is represented only as advisory mirrors; effective/public truth is pinned to broker authority.
- Audit modeling is intentionally minimal: required-obligation facts for the closed `v0` authoritative transition matrix.
- Broker-owned execution wait vocabulary is modeled separately from public run lifecycle. `waiting_approval` and `waiting_operator_input` are execution-scope wait states, not run lifecycle enum values.
- Partial blocking is modeled at execution-scope granularity: one scope may wait or block while unrelated eligible work remains runnable.
- Project-context-sensitive continuation is modeled as bound to a validated project-substrate digest and fails closed when the current digest drifts.

## Current Kernel Boundary

`SecurityKernelV0` is still intentionally bounded. It does not attempt to model the full product, transcript surfaces, or every later feature. It does now include the broker-owned execution semantics that later workflow and session changes made part of the shared kernel boundary:

- approval, plan, gate, and broker-authority invariants
- execution-scope wait vocabulary (`waiting_approval`, `waiting_operator_input`, and related wait kinds)
- dependency-aware partial blocking for downstream scopes
- validated project-substrate digest binding for project-context-sensitive execution

It still treats UI behavior, transport sequencing details, and broader workflow authoring as out of scope unless they affect these kernel contracts.

## Traceability Anchors

Primary references are encoded in comments at the end of the module and align to:

- `runecontext/changes/CHG-2026-015-cae6-formal-spec-v0-tla-ci-model-checking/design.md`
- `runecontext/standards/security/approval-binding-and-verifier-identity.md`
- `runecontext/standards/security/runner-durable-state-and-replay.md`
- `runecontext/standards/security/policy-evaluation-foundations.md`
- `runecontext/standards/security/trusted-runtime-evidence-and-broker-projection.md`
- `runecontext/standards/global/session-execution-contract-and-watch-families.md`
- `runecontext/standards/global/project-substrate-contract-and-lifecycle.md`
- related change designs: `CHG-2026-007`, `CHG-2026-008`, `CHG-2026-012`, `CHG-2026-033`, `CHG-2026-035`, `CHG-2026-046`, `CHG-2026-048`, `CHG-2026-050`

## Running TLC

TLC wiring is owned by the CI/tooling lane. When TLC tooling is available, run with this module and configs:

- module: `formal/tla/security-kernel/SecurityKernelV0.tla`
- configs:
  - `formal/tla/security-kernel/SecurityKernelV0.core.cfg`
  - `formal/tla/security-kernel/SecurityKernelV0.replay.cfg`
