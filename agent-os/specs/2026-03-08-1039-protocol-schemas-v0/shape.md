# Protocol & Schema Bundle v0 — Shaping Notes

## Scope

Define the schema foundation for manifests, artifacts, audit events, approvals, policy decisions, and the model-gateway request/response boundary.

## Decisions

- Cross-boundary communication is schema-validated and logged; no freeform text triggers privileged actions.
- No freeform prompt blobs cross trust boundaries; model requests reference artifacts by hash and remain typed (`LLMRequest`/`LLMResponse`).
- MVP model protocol supports streaming and tool calling, but only as typed artifacts/proposals (never direct execution).
- Structured JSON outputs are required for any machine-consumed output that can drive actions.
- Hashing/signing requires deterministic, canonical serialization (no "best effort" encoding).
  - Canonicalization is RFC 8785 JCS and is validated via cross-language golden fixtures.
- Signed objects use a standardized signature envelope including `{alg, key_id, signature}` (algorithm agility).
- Schemas carry field-level data classification metadata (`public | sensitive | secret`) to make redaction/boundary enforcement structural.
- MVP favors a single schema source of truth that can be validated in both Go and TS.
- MVP starts with JSON on-wire; the logical object model stays encoding-agnostic so on-wire RPC can migrate post-MVP to protobuf over local IPC (gRPC is optional and must remain local-only).
- Approvals are first-class, typed objects in the protocol (not ad-hoc prompts), enabling a future range of human-in-the-loop profiles.
- Post-MVP: add a schema-validated `ProcessDefinition` (JSON/YAML) to configure allowlisted step graphs without introducing new capabilities.
  - Adding allowlisted step types is treated as a capability expansion (schema bump + security review).
- Post-MVP: define a typed `bridge` provider envelope with an explicit "LLM-only" mode and a stable error taxonomy.

- A shared error taxonomy + envelope is part of MVP so daemons do not invent ad-hoc error shapes.

## Context

- Visuals: None.
- References: `agent-os/product/mission.md`, `agent-os/product/tech-stack.md`
- Product alignment: Enables auditable, policy-enforced boundaries between components/roles.

## Standards Applied

- None yet.
