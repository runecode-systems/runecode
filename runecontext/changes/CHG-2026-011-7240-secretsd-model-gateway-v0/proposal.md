## Summary
Track `secretsd` and `model-gateway` as a coordinated parent change while freezing the shared contracts that later auth, bridge, and provider features must reuse.

## Problem
Shared secret-management and model-egress work was previously tracked as one large feature, which reduced implementation and verification granularity.

Even after splitting the lane into child features, several foundation decisions remained implicit: the reusable secret-lease contract, auth versus model gateway separation, canonical destination identity, gateway operation vocabulary, request-hash binding, broker-visible posture surfaces, and quota semantics that must work across token-metered APIs and request-entitlement products. Leaving those decisions implicit would make later auth, bridge, and provider lanes more likely to drift while still appearing consistent at a high level.

## Proposed Change
- Keep this change as the parent project tracker for the lane.
- Track `CHG-2026-031-7a3c-secretsd-core-v0` as the secrets lifecycle and custody feature.
- Track `CHG-2026-032-4d1f-model-gateway-v0` as the canonical model egress boundary feature.
- Keep the following cross-feature foundation decisions explicit and aligned across both child features and later downstream lanes:
  - `secretsd` is the only long-lived credential store.
  - A typed `SecretLease` family is the reusable contract for stored secrets and future derived short-lived tokens.
  - `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` remain the canonical model request, response, and stream contracts.
  - `auth-gateway` and `model-gateway` remain separate least-privilege roles.
  - `destination_ref` uses one canonical host/port/path form rather than raw URLs.
  - gateway operations use one closed shared registry, and request-execution egress actions bind `payload_hash` to the canonical request object hash.
  - broker-projected subsystem posture remains the only long-lived operator-facing visibility surface for secrets and gateway readiness.
  - one trusted quota abstraction models provider request and token limits plus request-entitlement products such as premium requests.
- Keep cross-feature sequencing, standards, and verification notes reviewable in one place.

## Why Now
This work remains scheduled for v0.1.0-alpha.4, and the shared foundation must be explicit before downstream auth, bridge, and provider features depend on it.

Keeping those decisions at the parent-project level preserves roadmap traceability while allowing finer-grained feature implementation and verification.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Provider SDK payloads, daemon-private state, and transport bindings are implementation details, not the canonical control-plane contract source of truth.

## Out of Scope
- Runtime implementation details that belong in child feature changes.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Relaxing trust boundaries, local-only posture, or least-privilege role separation to simplify downstream provider integrations.
- Making raw provider payloads or daemon-private user APIs the canonical secrets or model-gateway contract.

## Impact
Keeps this lane reviewable as a parent project with explicit child features, clearer execution boundaries, and one stronger foundation for later auth, bridge, and provider work to build on without redefining core security semantics.
