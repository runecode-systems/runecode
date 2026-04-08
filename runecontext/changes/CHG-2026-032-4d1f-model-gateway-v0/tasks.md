# Tasks

## Gateway Boundary

- [ ] Implement an egress allowlisted model-gateway role.
- [ ] Ensure no workspace access and no long-lived secret storage.
- [ ] Model model-provider endpoints through the shared typed `DestinationDescriptor` / allowlist-entry pattern.

## Typed Model Contracts

- [ ] Enforce typed request/response boundaries.
- [ ] Ensure tool calls remain untrusted proposals.

## Egress Hardening

- [ ] Enforce destination validation and TLS requirements.
- [ ] Apply strict timeout and response-size limits.
- [ ] Keep redirect and destination validation semantics aligned with the shared policy-foundation gateway rules.

## Data Class + Policy

- [ ] Enforce allowlisted egress data classes.
- [ ] Block disallowed classes at the boundary.

## Audit + Quotas

- [ ] Audit outbound destination, bytes, timing, and outcome.
- [ ] Enforce basic quota controls.

## Acceptance Criteria

- [ ] Model egress occurs only through the gateway boundary.
- [ ] Gateway behavior is policy-controlled, auditable, and fail-closed.
