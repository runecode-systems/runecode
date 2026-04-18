# Tasks

## Local Bootstrap and Supervision

- [ ] Define a normal local product bootstrap flow for starting and supervising the trusted local control-plane services.
- [ ] Keep supervision and service orchestration local-only rather than exposing a second public lifecycle API.
- [ ] Preserve topology-neutral client contracts so later platform-specific service managers remain additive.

## Persistent Session Lifecycle

- [ ] Persist enough canonical session metadata to reconnect safely after TUI close or local process restart.
- [ ] Keep sessions and linked runs durable beyond the life of any one UI client.
- [ ] Define safe attach, detach, reconnect, and restart behavior in terms of broker-owned truth.

## Broker-Projected Posture

- [ ] Project blocked, degraded, and ready posture through broker-visible status rather than client-local guesswork.
- [ ] Keep readiness and version surfaces sufficient for attachable TUI and CLI clients.
- [ ] Keep daemon-private supervision health non-authoritative for operator UX.

## TUI and CLI UX

- [ ] Support attach/detach/reconnect flows in the TUI without making the TUI authoritative for lifecycle truth.
- [ ] Provide equivalent CLI entry and recovery flows as thin adapters over the same broker-owned model.

## Acceptance Criteria

- [ ] RuneCode can start as one coherent local product lifecycle and later reconnect to active sessions and runs.
- [ ] Sessions and runs continue safely when the TUI is closed.
- [ ] Broker remains the only public control-plane authority surface.
- [ ] Client-visible readiness and blocked-state posture are broker-projected and topology-neutral.
