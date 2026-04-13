# Tasks

## Bridge Runtime Contract

- [ ] Define shared bridge/runtime object families once for later provider specs.
- [ ] Keep bridge runtimes in explicit LLM-only mode with no workspace or patch capabilities.
- [ ] Keep bridge/runtime integrations below the canonical `LLMRequest` / `LLMResponse` / `LLMStreamEvent` model boundary.

## Compatibility + Probe Model

- [ ] Define probe-driven compatibility checks.
- [ ] Fail closed on unsupported or untested runtime versions instead of trusting newer vendor versions implicitly.

## Token Delivery + Session Rules

- [ ] Keep token delivery away from environment variables and raw secret logging.
- [ ] Define persisted-session posture and lifecycle rules explicitly.
- [ ] Use the canonical lease boundary for short-lived token handoff rather than bridge-local credential-delivery semantics.

## Audit + UX Surfaces

- [ ] Surface untested-version and persisted-session posture in audit and TUI flows.
- [ ] Keep bridge runtime behavior auditable and reviewable.

## Shared Foundation Inheritance

- [ ] Inherit canonical destination identity from the shared destination descriptor and `destination_ref` model.
- [ ] Inherit the shared trusted quota model rather than defining bridge-local usage accounting semantics.
- [ ] Keep any operator-facing posture broker-projected rather than exposing a second daemon-style public API.

## Acceptance Criteria

- [ ] Shared bridge contracts are reusable by provider-specific changes.
- [ ] Bridge runtimes remain LLM-only and fail closed on unsupported versions or unsafe token-delivery paths.
- [ ] Provider-specific bridge integrations cannot redefine the canonical model boundary, destination identity, or token-handoff semantics.
