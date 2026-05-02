## Summary
RuneCode can generate and verify one narrowly scoped local zero-knowledge proof that attests to a deterministic audit-bound integrity claim without revealing the full private session payload.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Deliver one narrow local audit-bound zero-knowledge integrity proof workflow rather than a broad proof lane for arbitrary reasoning or policy-program execution.
- Freeze the exact `v0` audited statement around one verified `isolate_session_bound` event with attested runtime bindings, so RuneCode proves one real high-value assurance seam first.
- Define a logical normalization profile and a scheme-adapter profile so the proof contract stays agnostic across `Groth16`, later `PLONK`, or a future zkVM-backed family.
- Keep proof generation explicit and opt-in; keep proof verification deterministic, trusted, and cheap enough for routine local use.
- Use a scheme-agnostic proof contract so future proof families may use a different proving system without rewriting RuneCode's broker, audit, or storage semantics.
- For `v0`, prefer a Go-native implementation path with `gnark` and a fixed-circuit `Groth16` verifier if the concrete performance gates are met.
- Make the first proof statement bind to canonical verified audit identity first, not to ambient repository state or an alternate product-private truth surface.
- Introduce `AuditProofBinding`-style proof-binding sidecars as part of the intended `v0` foundation so local proofs are reconstructed from canonical authoritative evidence rather than ambient local process state.
- Treat authoritative proof persistence for the first proof as audit-owned sidecar evidence; artifact-store copies remain optional review/export products rather than the primary trust source.
- Require every RuneCode machine to persist enough canonical proof-relevant source evidence and proof-binding information locally, even when no remote or public proof lane is enabled, so future backfill prerequisites are never lost.
- Bind any project-context-sensitive proof statement to the validated RuneContext project-substrate snapshot digest in verified mode rather than ambient repository assumptions.
- Bind any runtime-sensitive proof statement to the attested runtime identity seam rather than only to pre-attestation launch assumptions.
- Bind any audit-anchoring-sensitive proof statement to the canonical `AuditSegmentSeal` subject, authoritative anchor receipt identity, canonical target descriptor identity where external anchoring is involved, and preserved attestation or project-context references rather than flattened summaries.
- Keep future additive remote/public proof-lane design out of this `v0` implementation scope; that follow-on planning now lives in `CHG-2026-055-b7e4-additive-remote-public-proof-lane`.
- Keep the trusted local Groth16 backend hard-disabled until reviewed trusted setup assets are delivered through trusted assets; runtime deterministic setup generation on user machines remains out of policy.
- Re-enable the backend only after every statement-critical public field is cryptographically bound, trusted verifier posture comes only from reviewed local assets, proof verification validates the referenced `AuditProofBinding` plus authoritative source evidence, trusted Go validation mirrors protocol-schema bounds, audit recording failures are fail closed, and the documented performance gates are enforced by required CI or scheduled checks.

## Why Now
This work now lands in `v0.1.0-alpha.10` as a narrow parallel assurance lane, after signing, attestation, and external audit anchoring have stabilized enough to give the first proof statement durable typed claims to bind to.

Keeping it pre-beta but non-blocking lets RuneCode explore one real proof path without displacing the core usable product cut or forcing proof design onto provisional assurance objects that are still changing.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- The additive remote/public proof lane, public-assurance publication, or recursive aggregation work captured in `CHG-2026-055-b7e4-additive-remote-public-proof-lane`.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps ZK Proof v0 (One Narrow Proof + Verify) reviewable as a RuneContext-native local-proof change, anchors it to RuneCode's trusted audit and verified-substrate foundations, and removes the need for a second semantics rewrite later.

This change now also captures the local persistence requirement that every RuneCode instance must retain canonical proof-relevant source evidence strongly enough that later backfill remains possible even if the additive remote/public lane is not yet enabled on that machine.

It also records that the current backend remains intentionally fail closed until the remaining correctness, setup-integrity, authoritative-verification, validation-tightness, audit-recording, and performance-gate prerequisites are met.
