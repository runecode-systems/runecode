# Tasks

## Auth Gateway Role Contract

- [ ] Define the dedicated auth-gateway role and its limited public egress surface.
- [ ] Keep auth egress isolated from model egress and workspace access.
- [ ] Model auth-provider destinations through the shared typed `DestinationDescriptor` / allowlist-entry pattern.
- [ ] Keep auth-provider operations aligned with the shared closed gateway-operation vocabulary rather than provider-local ad hoc strings.
- [ ] Keep auth destination identity aligned with the shared logical destination-identity model rather than transport-URL-local policy decisions.

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
- [ ] Scope auth-derived leases to canonical destination identity, allowed operation set, and relevant action or policy bindings where supported by the shared lease model.

## Audit + Policy Integration

- [ ] Record typed auth lifecycle events without leaking secret values.
- [ ] Bind auth flows into policy and approval surfaces where required.
- [ ] Keep auth-gateway approval and error semantics aligned with shared `policy_reason_code`, `approval_trigger_code`, and system `error.code` ownership.

## Operator Posture

- [ ] Keep user-facing or operator-facing auth posture broker-projected rather than exposing a second daemon-specific public API.
- [ ] Add broker-owned typed setup and account-linking flows for auth posture and provider account state.
- [ ] Keep guided TUI setup and straightforward CLI setup as thin adapters over the same broker-owned typed auth flows.
- [ ] If manual token entry is ever required as a provider fallback, keep it limited to trusted interactive broker-mediated prompts rather than flags or environment variables.

## Acceptance Criteria

- [ ] Provider auth runs through an auth-only gateway role.
- [ ] Long-lived auth material remains isolated to `secretsd`.
- [ ] Auth failures remain typed, auditable, and fail closed.
- [ ] Short-lived auth-derived material reaches downstream consumers only through the canonical lease boundary.
- [ ] TUI and CLI setup surfaces remain thin clients of broker-owned typed auth flows, with no daemon-local setup authority.
