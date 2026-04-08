---
schema_version: 1
id: global/control-plane-api-contract-shape
title: Control-Plane API Contract Shape
status: active
suggested_context_bundles:
    - protocol-foundation
    - go-control-plane
---

# Control-Plane API Contract Shape

Boundary-visible control-plane APIs must keep their logical contract explicit, typed, and topology-neutral.

- Define operation-specific request and response object families under `protocol/schemas/`; do not treat ad-hoc JSON or transport-specific method envelopes as the contract source of truth.
- Keep public read models topology-neutral; do not require socket names, local usernames, daemon-private storage layouts, or host-local filesystem paths as part of boundary-visible object identity.
- Use shared typed error envelopes and stable reason-code registries for machine handling; do not rely on transport close, exit status, or scraped prose as the API error contract.
- Stream families must use explicit typed events with stable stream identity, monotonic sequence numbers, and exactly one terminal event.
- Use opaque cursor pagination and explicit ordering semantics for list and timeline operations; do not rely on page-number conventions or undocumented default sort behavior.
- Keep transport bindings, local IPC details, and CLI ergonomics as implementations of the logical API contract rather than the source of that contract.
