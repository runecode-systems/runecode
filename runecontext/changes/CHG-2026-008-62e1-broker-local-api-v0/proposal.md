## Summary
Isolates and clients communicate through a brokered, schema-validated, rate-limited local API with no isolate-to-isolate direct networking and with approval flows carried as typed signed artifacts rather than channel-specific trust shortcuts.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

## Proposed Change
- Define a transport-neutral logical broker API with protocol-owned typed request, response, read-model, and stream object families.
- Make run inspection first-class with `list runs` and `get run detail` surfaces that later TUI, CLI, concurrency, and remote/bridge work can build on without redefining run identity or lifecycle state.
- Keep run posture semantics explicit so broker read models do not collapse backend kind, runtime isolation assurance, provisioning posture, and audit posture into one ambiguous status field.
- Carry approval review and resolution through typed signed approval artifacts, canonical approval-request identity, structured bound-scope metadata, and explicit approval lifecycle states.
- Expose artifact and audit reads through broker-owned derived views that preserve trust boundaries and do not leak daemon-private storage internals.
- Keep local IPC auth, permissions, framing, limits, and rate posture explicit while preserving topology-neutral logical contracts for later alternate transports.

Core API foundation decisions for this change:
- The logical API is authoritative; Unix sockets, named pipes, JSON framing, and later protobuf are transport choices only.
- MVP uses operation-specific request/response families rather than a generic `method + any` envelope.
- Every request and response carries a stable `request_id` so tracing, audit, and future alternate transports remain aligned.
- Streaming operations use explicit typed stream-event families with `stream_id`, monotonic `seq`, and exactly one terminal event per stream.
- Failed terminal stream events carry the shared typed error envelope rather than transport-specific failure framing.
- Public broker read models stay operator-facing and topology-neutral; they do not expose host-local blob paths, socket names, usernames, or other transport/storage implementation details except optional diagnostics where explicitly allowed.
- Broker-visible state separates authoritative trusted or broker-derived status from runner-internal advisory state so the local API does not accidentally elevate untrusted orchestration details into trusted truth.
- Broker run surfaces must expose backend/runtime posture as distinct dimensions so later microVM/container, TUI, durable-state, and cross-platform changes can reuse one operator vocabulary.

## Why Now
This work remains scheduled for v0.1.0-alpha.3, and it is the narrowest point where RuneCode can set one durable control-plane API foundation before TUI, runner durable state, concurrency, and alternate transport work land.

If the logical broker API is left implicit or ad hoc now:
- the TUI will couple to daemon-private structs or CLI output
- the runner durable-state change will define its own run and approval vocabulary
- concurrency will need a second run-state model
- later protobuf transport work will end up migrating semantics rather than only encoding

Defining the logical contract here avoids a second semantics rewrite later while keeping the implementation local-first and reviewable.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Remote/network-facing client APIs.
- Multi-user or forwarded/proxied broker access.
- Generic event-bus or subscription APIs beyond the typed streams needed now.
- A second trust model for approvals based on UI clicks, delivery channels, or socket identity.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Keeps Broker + Local API v0 reviewable as a RuneContext-native change, gives upcoming changes a stable operator-facing logical contract, and removes the need for a second semantics rewrite later.
