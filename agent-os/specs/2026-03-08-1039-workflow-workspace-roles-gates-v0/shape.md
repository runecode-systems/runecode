# Workflow Runner + Workspace Roles + Deterministic Gates v0 — Shaping Notes

## Scope

Build the end-to-end workflow engine and offline workspace execution roles, with deterministic gates and evidence artifacts.

## Decisions

- The scheduler is treated as untrusted; the launcher/policy is the enforcement point.
- Workspace roles are offline; model egress (if enabled) is only via model-gateway.
- Pause/resume is implemented via a persisted run state machine (durable state), not in-memory orchestration.
- Gate failure semantics are explicit (fail/abort, retry, and any override requires recorded approval).

## Context

- Visuals: None.
- References: `agent-os/product/tech-stack.md`
- Product alignment: Spec-first, least-privilege automation with auditable evidence.

## Standards Applied

- None yet.
