## Summary
RuneCode can generate and verify at least one narrowly scoped zero-knowledge proof that attests to deterministic integrity claims without revealing sensitive contents.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Deliver one narrow audit-bound zero-knowledge integrity proof workflow rather than a broad proof lane for arbitrary reasoning or policy-program execution.
- Keep proof generation explicit and opt-in; keep proof verification deterministic, trusted, and cheap enough for routine local use.
- Use a scheme-agnostic proof contract so future proof families may use a different proving system without rewriting RuneCode's broker, audit, or storage semantics.
- For `v0`, prefer a Go-native implementation path with `gnark` and a fixed-circuit `Groth16` verifier if the concrete performance gates are met.
- Make the first proof statement bind to canonical verified audit identity first, not to ambient repository state or an alternate product-private truth surface.
- Treat authoritative proof persistence for the first proof as audit-owned sidecar evidence; artifact-store copies remain optional review/export products rather than the primary trust source.
- Bind any project-context-sensitive proof statement to the validated RuneContext project-substrate snapshot digest in verified mode rather than ambient repository assumptions.
- Bind any runtime-sensitive proof statement to the attested runtime identity seam rather than only to pre-attestation launch assumptions.
- Bind any audit-anchoring-sensitive proof statement to the canonical `AuditSegmentSeal` subject, authoritative anchor receipt identity, canonical target descriptor identity where external anchoring is involved, and preserved attestation or project-context references rather than flattened summaries.

## Why Now
This work now lands in `v0.1.0-alpha.10` as a narrow parallel assurance lane, after signing, attestation, and external audit anchoring have stabilized enough to give the first proof statement durable typed claims to bind to.

Keeping it pre-beta but non-blocking lets RuneCode explore one real proof path without displacing the core usable product cut or forcing proof design onto provisional assurance objects that are still changing.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps ZK Proof v0 (One Narrow Proof + Verify) reviewable as a RuneContext-native change, anchors it to RuneCode's trusted audit and verified-substrate foundations, and removes the need for a second semantics rewrite later.
