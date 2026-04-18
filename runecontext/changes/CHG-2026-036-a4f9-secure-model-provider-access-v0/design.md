# Design

## Overview
Use a project-level change to coordinate secure model/provider access features and shared trust-boundary requirements.

## Key Decisions
- Shared security invariants apply to all child features.
- Secrets lifecycle, auth, bridge, and provider lanes remain separable feature boundaries.
- Verification remains feature-level, with this project change tracking sequencing and integration posture.
- Direct-credential, OAuth, and bridge-runtime provider lanes should share one provider-profile and auth-material substrate.

## Shared Inherited Foundation

- `secretsd` remains the only long-lived credential store.
- `SecretLease` remains the canonical short-lived secret and token handoff contract.
- `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` remain the canonical model request, response, and stream families.
- Provider profile, auth-material, readiness, and compatibility posture should be shared across direct-credential and later auth/bridge provider lanes.
- `auth-gateway` and `model-gateway` remain distinct least-privilege roles.
- Canonical destination identity and gateway operations are shared and closed rather than provider-local ad hoc semantics.
- Request-execution egress actions should bind to canonical request identity rather than to transport-local or provider-local payload shapes.
- Operator-facing posture remains broker-projected.
- Quota handling remains one trusted abstraction that can represent token-metered APIs, concurrency limits, spend ceilings, and request-entitlement products.

## Sequencing Rules

- Direct-credential providers should land before later OAuth and bridge-runtime providers, but they must do so on the same provider-profile and auth-material substrate.
- Provider-specific lanes should inherit, not redefine, the reviewed `secretsd` and model-gateway foundation.
- Auth and bridge lanes should become the only provider-specific places where OAuth/runtime compatibility details live; they should not re-open core secret-custody or canonical model-boundary decisions.
- Provider-specific features should stay downstream of the shared destination-identity and quota model so each provider does not invent its own egress identity or usage-accounting semantics.

## Main Workstreams
- Shared foundation tracking (`secretsd` and model-gateway).
- Direct-credential provider feature sequencing.
- Auth and bridge feature sequencing.
- Provider-specific feature sequencing and integration checks.
- Cross-lane inherited contract review.
