# Tasks

## Official Runtime Bridge Integration

- [ ] Integrate the official local Copilot runtime bridge through the shared bridge/runtime protocol.
- [ ] Keep the runtime in explicit LLM-only mode with no workspace or patch capabilities.

## Auth-Gateway + Token Delivery

- [ ] Route auth through the auth-gateway role.
- [ ] Keep long-lived auth material isolated to `secretsd`.
- [ ] Avoid environment-variable token delivery and raw secret logging.
- [ ] Keep provider setup and account-linking on broker-owned typed flows surfaced through guided TUI and straightforward CLI clients.

## Compatibility + Probe Policy

- [ ] Keep runtime compatibility probe-driven and fail-closed.
- [ ] Surface unsupported or untested runtime posture clearly.

## Policy + Audit Integration

- [ ] Keep provider enablement as an explicit approved posture change.
- [ ] Record auth and model egress events without expanding the trust boundary.
- [ ] Keep provider account and auth posture broker-projected rather than runtime-local.

## Acceptance Criteria

- [ ] Copilot model access stays behind the shared bridge/runtime and auth-gateway model.
- [ ] The provider remains LLM-only, auditable, and fail-closed on unsupported runtime posture.
