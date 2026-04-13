# Tasks

## Gateway Boundary

- [ ] Implement an egress allowlisted model-gateway role.
- [ ] Ensure no workspace access and no long-lived secret storage.
- [ ] Model provider endpoints through the shared typed `DestinationDescriptor` / allowlist-entry pattern.
- [ ] Keep auth-gateway and model-gateway separation explicit so model traffic never performs auth-provider exchange or refresh in place.

## Destination + Operation Identity

- [ ] Freeze one canonical `destination_ref` form based on host/port/path identity rather than raw URLs.
- [ ] Introduce or adopt one closed gateway operation vocabulary shared with the broader gateway policy foundation.
- [ ] Distinguish scope-change operations from request-execution operations.

## Typed Model Contracts

- [ ] Enforce typed request/response boundaries.
- [ ] Ensure tool calls remain untrusted proposals.
- [ ] Keep `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` as the canonical typed model boundary rather than provider-specific payloads.
- [ ] Require request-execution gateway actions to bind `payload_hash` to the canonical `LLMRequest` hash.

## Egress Hardening

- [ ] Enforce destination validation and TLS requirements.
- [ ] Apply strict timeout and response-size limits.
- [ ] Keep redirect and destination validation semantics aligned with the shared policy-foundation gateway rules.
- [ ] Keep private-range blocking and DNS rebinding protection aligned with the shared destination descriptor invariants.

## Data Class + Policy

- [ ] Enforce allowlisted egress data classes.
- [ ] Block disallowed classes at the boundary.

## Audit + Quotas

- [ ] Audit outbound destination, bytes, timing, and outcome.
- [ ] Bind audit records to canonical request, response, lease, and policy identity where applicable.
- [ ] Enforce quota controls using one trusted abstraction that can represent token-metered APIs and request-entitlement products.
- [ ] Apply quota handling at admission time and during streaming where limits require it.

## Operator Posture

- [ ] Keep daemon-local health/readiness supervision local-only.
- [ ] Keep user-facing or operator-facing model-gateway posture broker-projected rather than exposing a second daemon-specific public API.

## Acceptance Criteria

- [ ] Model egress occurs only through the gateway boundary.
- [ ] Gateway behavior is policy-controlled, auditable, request-bound, and fail-closed.
- [ ] Downstream provider features can reuse the gateway boundary without redefining destination identity, operation semantics, or quota handling.
