## Summary
Track secure model and provider integration as one project-level plan covering shared secret lifecycle, canonical model egress boundaries, direct-credential provider access, later auth flows, bridge contracts, provider-specific feature lanes, and the cross-feature contracts they must all inherit.

## Problem
Provider integration work is currently spread across multiple feature changes without a single project-level tracker for sequencing and verification.

Now that the `secretsd` and `model-gateway` foundation has been refined, the provider umbrella also needs to state those shared contracts explicitly across both the pre-beta direct-credential lane and the later OAuth and bridge lanes. Otherwise provider work could still drift on shared provider-profile shape, auth-material handling, canonical request boundaries, destination identity, operator posture, or quota semantics even while claiming to preserve the same high-level trust model.

## Proposed Change
- Keep a project-level tracker for secure model/provider integration.
- Link shared foundation features and provider-specific features under one change.
- Track `CHG-2026-045-7f4c-direct-credential-model-providers-v0` as the pre-beta direct-credential provider lane.
- Preserve strict trust-boundary assumptions across all child features.
- Freeze the inherited cross-feature foundation that downstream auth, bridge, and provider lanes must reuse:
  - `secretsd` is the only long-lived credential store.
  - `SecretLease` is the canonical short-lived secret and token handoff contract.
  - `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` remain the canonical model boundary.
  - one shared provider-profile and auth-material substrate is reused by direct-credential, OAuth, and bridge-runtime provider lanes.
  - `auth-gateway` and `model-gateway` remain distinct least-privilege roles.
  - `destination_ref` and gateway operations use shared canonical identity rather than provider-local URL or method semantics.
  - request-execution egress actions bind to canonical request identity.
  - operator-facing posture is broker-projected.
  - quota handling uses one trusted abstraction that can represent both token-metered APIs and request-entitlement products.

## Why Now
The provider lane spans multiple releases and shared foundations; a project-level change improves visibility, sequencing, and verification discipline.

This umbrella should now also record the pre-beta direct-credential lane on the same durable provider substrate so later OAuth and bridge-provider changes inherit that foundation explicitly instead of quietly narrowing or redefining it.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Provider-specific runtime protocols and SDK payloads remain implementation details below the shared trusted contracts unless a later typed extension is reviewed and made explicit.

## Out of Scope
- Implementing provider feature runtime behavior directly in this project tracker.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Letting provider lanes redefine secret custody, model-boundary, destination-identity, or quota semantics independently.

## Impact
Creates a project-level anchor for provider and model-access sequencing while leaving delivery details in feature changes and making the inherited shared security contracts explicit for both the pre-beta direct-credential lane and later provider work.
