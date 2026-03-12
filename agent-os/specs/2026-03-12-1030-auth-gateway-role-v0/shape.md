# Auth Gateway Role v0 — Shaping Notes

## Scope

Define the provider-agnostic `auth-gateway` role boundary and contracts for OAuth/device-code flows, secrets storage, and audit.

## Decisions

- Auth egress and model egress are separated:
  - `auth-gateway` performs login/refresh and has auth-only egress.
  - `model-gateway` performs inference egress and never receives long-lived credentials.
- `secretsd` is the only long-lived secrets store; there is no second credential cache.
- No environment-variable or CLI-arg secret injection.
- Auth flows are typed, auditable, and fail closed on state/protocol mismatches.

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-secretsd-model-gateway-v0/`, `agent-os/specs/2026-03-08-1039-policy-engine-v0/`
- Product alignment: Preserves least-privilege by isolating public auth egress away from workspace access.

## Standards Applied

- None yet.
