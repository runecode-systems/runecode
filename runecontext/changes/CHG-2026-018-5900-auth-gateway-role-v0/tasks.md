# Tasks

## Auth Gateway Role Contract

- [ ] Define the dedicated auth-gateway role and its limited public egress surface.
- [ ] Keep auth egress isolated from model egress and workspace access.
- [ ] Model auth-provider destinations through the shared typed `DestinationDescriptor` / allowlist-entry pattern.
- [ ] Keep auth-provider operations aligned with the shared closed gateway-operation vocabulary rather than provider-local ad hoc strings.

## Provider-Agnostic Auth Objects

- [ ] Define shared typed auth object families and versioned control flow.
- [ ] Ensure provider-specific flows extend the shared objects instead of redefining core semantics.
- [ ] Ensure typed auth request-execution operations can bind to canonical request identity instead of relying on transport or provider-local payload shapes.

## Secret Handling + Token Storage

- [ ] Keep `secretsd` as the only long-lived secrets store.
- [ ] Forbid environment-variable or CLI-arg secret injection.
- [ ] Keep login, refresh, and token delivery flows typed and fail closed.
- [ ] Use the canonical `SecretLease` family for short-lived token handoff to downstream consumers such as `model-gateway`.
- [ ] Keep `model-gateway` from performing login, token exchange, or refresh in place.

## Audit + Policy Integration

- [ ] Record typed auth lifecycle events without leaking secret values.
- [ ] Bind auth flows into policy and approval surfaces where required.
- [ ] Keep auth-gateway approval and error semantics aligned with shared `policy_reason_code`, `approval_trigger_code`, and system `error.code` ownership.

## Operator Posture

- [ ] Keep user-facing or operator-facing auth posture broker-projected rather than exposing a second daemon-specific public API.

## Acceptance Criteria

- [ ] Provider auth runs through an auth-only gateway role.
- [ ] Long-lived auth material remains isolated to `secretsd`.
- [ ] Auth failures remain typed, auditable, and fail closed.
- [ ] Short-lived auth-derived material reaches downstream consumers only through the canonical lease boundary.
