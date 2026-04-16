# Security Kernel TLA+ Spec (CHG-2026-015)

This directory introduces the first checked-in TLA+ artifacts for the bounded workflow security kernel.

## Files

- `SecurityKernelV0.tla` — bounded kernel model of runs, plans, stage summaries, action requests, approvals, gate attempts/evidence, broker-vs-runner authority, and minimal audit obligations.
- `SecurityKernelV0.core.cfg` — deterministic TLC model focused on core approval/stage/gate/lifecycle invariants.
- `SecurityKernelV0.replay.cfg` — deterministic TLC model that keeps replay/reconciliation paths in scope with slightly broader bounds.
- `SecurityKernelV0_MC.cfg` — compatibility config kept alongside the dedicated model configs.

## Scope Notes

- Opaque deterministic tokens are modeled as bounded sets (`HashTokens`, `ArtifactDigests`, `PolicyHashes`, `PolicyInputHashes`, `EvidenceRefs`) instead of byte-level hashing.
- Stage sign-off semantics are bound to canonical `StageSummary` identity via `stageSummaryHash` (not `RunStageSummary`).
- Runner durable state is represented only as advisory mirrors; effective/public truth is pinned to broker authority.
- Audit modeling is intentionally minimal: required-obligation facts for the closed `v0` authoritative transition matrix.

## Traceability Anchors

Primary references are encoded in comments at the end of the module and align to:

- `runecontext/changes/CHG-2026-015-cae6-formal-spec-v0-tla-ci-model-checking/design.md`
- `runecontext/standards/security/approval-binding-and-verifier-identity.md`
- `runecontext/standards/security/runner-durable-state-and-replay.md`
- `runecontext/standards/security/policy-evaluation-foundations.md`
- `runecontext/standards/security/trusted-runtime-evidence-and-broker-projection.md`
- related change designs: `CHG-2026-007`, `CHG-2026-008`, `CHG-2026-012`, `CHG-2026-033`, `CHG-2026-035`

## Running TLC

TLC wiring is owned by the CI/tooling lane. When TLC tooling is available, run with this module and configs:

- module: `formal/tla/security-kernel/SecurityKernelV0.tla`
- configs:
  - `formal/tla/security-kernel/SecurityKernelV0.core.cfg`
  - `formal/tla/security-kernel/SecurityKernelV0.replay.cfg`
