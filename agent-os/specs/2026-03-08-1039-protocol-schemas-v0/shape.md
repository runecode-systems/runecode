# Protocol & Schema Bundle v0 — Shaping Notes

## Scope

Define the schema foundation for manifests, artifacts, audit events, approvals, and policy decisions.

## Decisions

- Cross-boundary communication is schema-validated and logged; no freeform text triggers privileged actions.
- Hashing/signing requires deterministic, canonical serialization (no "best effort" encoding).
- Schemas carry field-level data classification metadata (`public | sensitive | secret`) to make redaction/boundary enforcement structural.
- MVP favors a single schema source of truth that can be validated in both Go and TS.
- MVP starts with JSON on-wire; the logical object model stays encoding-agnostic so on-wire RPC can migrate post-MVP to protobuf over local IPC (gRPC is optional and must remain local-only).

## Context

- Visuals: None.
- References: `agent-os/product/mission.md`, `agent-os/product/tech-stack.md`
- Product alignment: Enables auditable, policy-enforced boundaries between components/roles.

## Standards Applied

- None yet.
