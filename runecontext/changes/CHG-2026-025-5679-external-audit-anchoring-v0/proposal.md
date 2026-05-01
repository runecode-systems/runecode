## Summary
RuneCode can optionally anchor audit roots to external targets with explicit egress, typed receipts, and the same shared gateway, approval, lease, and audit-evidence discipline used for other high-risk remote mutation lanes.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Later Anchor Target Model.
- Egress + Trust Boundary Model.
- Receipt, Audit, and Verification Integration.
- Fixtures + Adapter Conformance.
- Explicit alignment with shared remote-state-mutation gateway semantics where external anchoring mutates remote target state.
- Explicit preservation of attestation evidence and verification references when anchored audit chains include attested runtime evidence.
- Exact-action approval as the `v0` outbound submission posture, with signed-manifest automation reserved as a near-term additive follow-on over the same typed request path.
- A durable prepared and execute model that supports deferred completion, bounded worker concurrency, and the same overall architecture from constrained local devices to scaled deployments.
- A typed target-descriptor authority model where target identity is bound by canonical descriptor digest rather than raw URL strings.
- An additive receipt and sidecar evidence model that keeps shared `AuditReceipt(kind=anchor)` small while preserving target-specific proof as typed authoritative sidecar evidence.
- A target-set foundation that supports later multi-target profiles while keeping `v0` runtime scope narrow with one concrete target family first.

## Why Now
This work now lands in `v0.1.0-alpha.10`, because the first usable release should include the full planned assurance trio: signed runtime identity, isolate attestation, and externally verifiable audit anchoring.

The git-gateway foundation now clarifies how other high-risk outbound lanes should talk about target identity, exact-action approval, lease scope, and shared audit proof, and external anchoring should inherit those decisions before the beta cut instead of drifting into an ad hoc egress model.

The first runtime delivery also needs to freeze the performance-sensitive foundation now so later automation, target expansion, and larger deployment footprints extend one efficient architecture instead of introducing separate small-device and scaled-deployment paths.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- The same overall trusted architecture must remain viable from low-resource single-node environments through vertically and horizontally scaled deployments; performance work should improve the shared path rather than split the product into topology-specific variants.
- The first concrete external target kind can be narrower than the full future target family set so long as the shared contracts remain additive and future-safe.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Signed-manifest automation for external anchor submission in the first implementation tranche.
- Requiring a quorum or partial-success target-set policy in `v0`; those remain additive follow-on semantics after the required-target foundation is frozen.

## Impact
Keeps External Audit Anchoring v0 reviewable as a RuneContext-native change, aligned with the reviewed shared gateway remote-mutation foundation, and removes the need for a second semantics rewrite later.

Freezing the approval, target-identity, execution-lifecycle, and evidence-binding foundations now also prevents a later performance rewrite by ensuring that background automation, new target kinds, and larger workloads all reuse the same typed broker, auditd, verifier, and gateway contracts.
