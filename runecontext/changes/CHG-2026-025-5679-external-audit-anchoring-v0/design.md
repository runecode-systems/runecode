# Design

## Overview
Define typed external anchoring targets, receipt verification, and explicit egress controls for non-local audit anchoring while preserving the shared `AuditReceipt(kind=anchor)` envelope, `AuditSegmentSeal` subject model, sidecar-authoritative evidence posture, and existing verifier status dimensions established by `CHG-2026-006-84f0-audit-anchoring-v0/`.

## Key Decisions
- Later non-MVP anchor targets use typed descriptors and typed anchor receipt payloads on the shared `AuditReceipt(kind=anchor)` envelope rather than creating a second top-level external-only receipt family.
- External anchoring is explicit opt-in with a clear egress model.
- `AuditSegmentSeal` remains the canonical anchoring subject for external targets just as it does for local anchors.
- External anchor receipts remain authoritative as sidecar audit evidence first; artifact-store copies remain optional export/review products.
- Verification/reporting must distinguish valid external anchors from deferred, unavailable, or invalid states using the shared verifier posture model rather than a separate external-only status taxonomy.
- External anchor submissions that mutate remote target state should align with the shared gateway remote-state-mutation class rather than an ad hoc outbound category.
- Target identity, approval scope, lease scope, and audit evidence for external anchoring should use the same reviewed shared-gateway discipline now frozen for git remote mutation.
- `v0` external anchor submission is an exact-action approval boundary for each outbound submission when remote target state is mutated; signed-manifest automation is a later additive posture over the same typed request path.
- The first runtime adapter should be one concrete target family, with transparency-log style anchoring as the recommended first implementation, while timestamp-authority and public-chain families remain typed follow-on work.
- External anchoring should use a durable prepared and execute lifecycle that allows inline completion when fast but treats deferred execution as a normal first-class outcome.
- The authoritative target identity should be a typed target descriptor bound by canonical descriptor digest rather than raw transport URL policy.
- The shared anchor receipt should stay minimal and additive, while target-specific external proof bytes remain in typed authoritative sidecar evidence referenced by digest.
- Performance improvements must preserve one architecture across constrained and scaled environments: no network I/O under audit ledger locks, no separate topology-specific control-plane path, and no full verifier replay as the only receipt-admission path.
- The foundation should support target sets, but aggregate success semantics for required targets should be `all required targets satisfied`; quorum-style policies are later additive work.

## Shared Gateway Alignment

- External anchoring should use typed target identity with exact matching rather than raw URL-only policy.
- If a target requires authenticated remote mutation, short-lived credentials must remain lease-bound through the shared secrets model rather than target-local secret handling.
- `v0` uses exact-action approval for each outbound external anchor submission rather than approved ambient automation.
- Later signed-manifest automation, if enabled, should call the same prepared and execute path and bind the same canonical request hashes, allowlist identity, policy decision identity, and lease scope rather than creating a second automation-only route.
- Whatever posture is chosen, it must be expressed through the shared policy and approval vocabulary rather than a target-local approval model.
- When `CHG-2026-059-7b31-cross-machine-evidence-replication-restore-v0` is active, external anchor submission should remain compatible with the shared durability-posture model for publication-sensitive remote mutation lanes: healthy evidence durability preconditions may gate submission execution, but no target-local lower-assurance exception path may bypass the shared trusted recovery model.
- Audit evidence should include canonical target identity, anchoring subject identity, outbound payload or subject hash, bytes, timing, outcome, and any lease or policy bindings needed to verify the outbound action.
- Where the anchored audit chain includes launch-admission, launch-deny, or other runtime evidence derived from the signed runtime-asset pipeline, external anchoring should preserve those runtime-identity references rather than flattening them into target-local summaries.
- Where the anchored audit chain includes attested isolate-session evidence, external anchoring should preserve attestation evidence and verification references rather than flattening them into target-local summaries or collapsing them into launch-only identity.
- When the anchored subject depends on project context, external anchoring should reuse the validated project-substrate snapshot identity already bound into the underlying audit evidence rather than inventing a second project-context reference.

## Canonical Target And Request Model

- External anchoring should introduce a provider-neutral typed target descriptor family for external anchor targets rather than treating `destination_ref` strings as the full authority surface.
- The authoritative identity used by policy, approval, lease binding, audit, and verification should be the canonical target descriptor digest.
- Transport-specific fields such as endpoint host, path, chain RPC endpoint, or TSA submission URL remain derived execution details below the typed target descriptor.
- The shared gateway path should add a provider-neutral typed request family for external anchor submission rather than reusing a local-only ad hoc anchor action hash.
- The typed request should bind at least:
  - the targeted `AuditSegmentSeal` digest
  - the canonical target descriptor digest
  - the canonical outbound subject or payload hash
  - the target kind and request kind
  - any target-specific immutable request fields that participate in exact-action approval and runtime verification
- The authoritative request hash for policy and approval should be the canonical RFC 8785 JCS hash of the typed external anchor request object.

## Receipt, Proof, And Evidence Model

- `AuditReceipt(kind=anchor)` remains the only canonical anchor receipt family.
- `AuditSegmentSeal` remains the only canonical anchor subject family for external targets just as it is for local targets.
- External-target-specific proof material should not bloat the shared signed receipt envelope with large provider-specific payloads when those bytes can instead live in authoritative digest-addressed sidecar evidence.
- The shared anchor receipt should carry the minimum typed fields needed to prove target kind, target identity binding, receipt schema binding, and proof reference identity.
- Target-specific proof bytes, verification transcripts, or returned provider receipts should live in typed sidecar evidence records referenced by digest from the anchor receipt or related audit evidence.
- Exported artifact copies remain optional review products and must remain copies of authoritative sidecar evidence rather than a second trust source.

## Execution Lifecycle And Concurrency Model

- External anchoring should use a durable broker-owned prepared and execute lifecycle rather than a request model that assumes every remote submission completes inline.
- The execution model should support `prepare`, `get`, and `execute` request families, with watch or polling-friendly read models for deferred completion.
- `execute` may return completed for fast targets, but deferred must be treated as a normal first-class result rather than an exceptional path.
- Durable attempt identity should be idempotent over the immutable request inputs, including the seal digest, canonical target descriptor identity, and typed request hash.
- Trusted services should snapshot immutable anchoring inputs under the audit ledger lock, release the lock before any network I/O or external wait, and reacquire only for final compare-and-persist work.
- External network I/O, target polling, and backoff handling must not occur while holding the audit ledger mutex.
- Bounded worker concurrency, durable retries, and explicit backoff should scale the shared path without changing the trust model between small devices and larger deployments.

## Verification And Performance Posture

- Trusted Go verification remains authoritative.
- The verifier must keep the shared fail-closed posture for invalid receipts and invalid sidecar proof while distinguishing missing or deferred required targets as degraded posture.
- External anchoring should not require a full segment replay on every successful receipt admission when the existing verified segment and seal state are unchanged.
- The foundation should support incremental receipt admission and per-seal verification snapshots so external anchoring scales with receipt count without replaying all segment verification work each time.
- Full recomputation remains available as the authoritative recovery and audit check path, but it should not be the only hot-path mechanism for each external anchor submission.
- Multiple receipts or retries for the same seal and target must not cause unbounded verifier work or quadratic growth in ordinary anchoring operations.

## Target-Set Semantics

- The foundation should support later target sets even if the first runtime implementation ships one concrete target family.
- Aggregate anchoring posture should be derived from required-target semantics rather than from ad hoc prose labels.
- Recommended aggregate semantics:
  - `ok`: every required target has valid external anchor evidence, and no authoritative persisted receipt for the anchored subject is invalid
  - `degraded`: no invalid authoritative evidence exists, but one or more required targets are deferred, unavailable, or not yet satisfied
  - `failed`: any required target produces invalid evidence, or any authoritative persisted receipt for the anchored subject is invalid
- Optional supplemental targets may remain visible in per-target findings without blocking aggregate `ok` once the required target set is satisfied.
- Quorum-style policies such as `min_valid_required` remain later additive work and should not be implied by the `required` vocabulary in `v0`.

## Main Workstreams
- Later Anchor Target Model
- Egress + Trust Boundary Model
- Receipt, Audit, and Verification Integration
- Fixtures + Adapter Conformance

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
