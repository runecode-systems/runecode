# Broker + Local API v0 — Shaping Notes

## Scope

Implement the brokered communication hub and the local API surface used by clients and the scheduler.

## Decisions

- Star topology only; no direct isolate-to-isolate communication.
- Schema validation at boundaries; rate limiting and backpressure are mandatory.
- Broker enforces concrete default limits (message size/complexity/in-flight/streaming) with audited overrides.
- The local API is per-user IPC with strict filesystem permissions; authentication fails closed when OS peer credentials are unavailable.
- Errors use a shared typed envelope and stable reason codes (no ad-hoc error shapes).
- MVP uses JSON on-wire; post-MVP may adopt protobuf over local IPC. gRPC is optional and must remain local-only (no TCP-by-default).

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`
- Product alignment: Supports least-privilege communication and auditability.

## Standards Applied

- None yet.
