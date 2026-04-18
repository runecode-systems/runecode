## Summary
RuneCode can start, supervise, and reconnect to its local control plane as one product lifecycle while sessions and runs persist beyond the life of the TUI.

## Problem
The current foundation already has trusted local services and a TUI client, but it still reads like a collection of components instead of one attachable local product. A user should not need to keep the TUI open to preserve work, and they should not have to manually reconstruct local service state after routine restarts.

Without an explicit feature, the system risks either staying too manual for normal use or introducing daemon-private lifecycle truth that bypasses the broker and existing typed contracts.

## Proposed Change
- Local control-plane bootstrap and supervision entry flows.
- Persistent session lifecycle and reconnect semantics.
- Broker-projected readiness and degraded-state posture for attachable clients.
- TUI and CLI attach/detach flows that remain thin clients of broker-owned state.

## Why Now
This work now lands in `v0.1.0-alpha.7`, immediately after direct model access and verified project substrate work, because the next user-facing step is making RuneCode behave like one coherent local product rather than a manual component assembly.

Landing lifecycle and reconnect semantics before interactive execution also avoids a later split between "chat mode while the TUI is open" and "background product mode".

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Broker remains the only public control-plane surface even when local bootstrap and supervision behavior grows.

## Out of Scope
- Remote or multi-host control-plane topologies.
- Re-introducing legacy Agent OS planning paths as canonical references.
- A second daemon-private user API for local lifecycle truth.

## Impact
Turns the current trusted local components into an attachable product lifecycle where work can continue without the TUI staying open, while preserving broker-owned truth and typed read/write contracts.
