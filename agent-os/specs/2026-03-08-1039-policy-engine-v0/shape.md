# Policy Engine v0 — Shaping Notes

## Scope

Implement the core policy evaluator that enforces manifests, role invariants, and explicit approvals.

## Decisions

- Deny-by-default everywhere; allow only via signed manifest.
- No automatic fallback to containers; container mode is explicit opt-in.
- MVP policy language is declarative and schema-validated (no general-purpose code execution during evaluation).

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`
- Product alignment: Centralizes least-privilege enforcement and “no escalation-in-place”.

## Standards Applied

- None yet.
