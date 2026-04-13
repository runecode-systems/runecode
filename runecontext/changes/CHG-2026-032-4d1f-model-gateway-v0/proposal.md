## Summary
RuneCode routes all third-party model traffic through a hardened gateway with one canonical typed model boundary, explicit destination and operation identity, data-class controls, request-hash binding, and auditable quota enforcement.

## Problem
The previous combined change coupled secrets foundations and model egress details, reducing review focus for network-bound trust boundary controls.

After the split, several gateway-defining choices still needed to be frozen explicitly: how destination identity is represented, how model operations are named, how actual request execution binds to typed request objects, how broker-visible posture differs from daemon-local supervision, and how quota controls should work across token-metered APIs and request-entitlement products.

## Proposed Change
- Dedicated `model-gateway` role with allowlisted egress.
- Keep `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` as the canonical typed model boundary.
- Require one canonical `destination_ref` identity form and one closed model-gateway operation vocabulary.
- Bind request-execution egress actions to canonical request object hashes through `payload_hash`.
- Enforce data-class policy and redirect-safe destination validation at the boundary.
- Enforce audit and quota controls for outbound model traffic.
- Keep user-facing or operator-facing posture broker-projected rather than exposing a second daemon-specific public API.

## Why Now
This feature keeps egress-bound controls independently reviewable while still aligning under the secure model/provider access project, and it freezes the shared gateway contracts before auth, bridge, and provider lanes depend on them.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Provider SDK payloads and bridge/runtime adapter details remain implementation details unless later typed extensions are required for policy, audit, or replay semantics.

## Out of Scope
- Secret storage internals and key posture recording.
- Provider-specific runtime bridge contracts.
- Auth-provider login or refresh control flows.
- Replacing the canonical model boundary with provider-specific request and response payloads.

## Impact
Keeps model egress controls and trust-boundary hardening reviewable as a standalone feature while providing a stronger reusable gateway contract for later auth, bridge, and provider work.
