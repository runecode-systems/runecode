# Tasks

## Shared Provider Substrate

- [ ] Define a shared provider-profile model that direct-credential, OAuth, and bridge-runtime lanes can all reuse.
- [ ] Define a shared auth-material model that distinguishes direct credentials from later derived or session-bound material without changing provider identity or readiness semantics.
- [ ] Keep provider-specific wire payloads below the canonical typed model boundary.
- [ ] Reuse shared destination identity, request binding, quota, and broker-projected posture semantics instead of introducing provider-local variants.

## Direct Credential Setup

- [ ] Support trusted interactive setup for operator-entered endpoint configuration and API credentials.
- [ ] Store long-lived direct credentials only in `secretsd`.
- [ ] Lease scope-bound credential material to `model-gateway` for request execution.
- [ ] Forbid environment-variable and command-line secret injection.

## Adapter Families

- [ ] Add an OpenAI-compatible adapter family beneath the canonical model boundary.
- [ ] Add an Anthropic-compatible adapter family beneath the canonical model boundary.
- [ ] Keep auth-mode-specific behavior separate from shared provider-profile, readiness, and audit semantics.

## Broker, TUI, and CLI Surfaces

- [ ] Add broker-owned setup, inspection, readiness, and compatibility surfaces for direct-credential providers.
- [ ] Surface supported auth modes and current compatibility posture in TUI and CLI flows.
- [ ] Keep provider setup, readiness, and error posture broker-projected rather than daemon-private.

## Acceptance Criteria

- [ ] RuneCode can use operator-entered OpenAI-compatible and Anthropic-compatible credentials for remote model access.
- [ ] Direct credentials reuse the same provider substrate that later OAuth and bridge-runtime features will extend.
- [ ] `secretsd` remains the only long-lived credential store and `model-gateway` remains the canonical model-egress lane.
- [ ] Canonical typed model contracts, shared destination identity, quota handling, and broker-projected readiness remain unchanged by auth mode.
