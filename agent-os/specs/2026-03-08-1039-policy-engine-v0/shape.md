# Policy Engine v0 — Shaping Notes

## Scope

Implement the core policy evaluator that enforces manifests, role invariants, and explicit approvals.

## Decisions

- Deny-by-default everywhere; allow only via signed manifest.
- No automatic fallback to containers; container mode is explicit opt-in.
- MVP policy language is declarative and schema-validated (no general-purpose code execution during evaluation).
- Core security invariants are non-negotiable; any "approval policy" or UX setting may only tighten policy, never loosen it.
- Network egress is a hard boundary: workspace roles are offline; public egress is only via explicit gateway roles (model inference via `model-gateway`), and non-gateway network egress is not approvable.
- MVP uses checkpoint-style approvals (stage sign-off and explicit posture changes) instead of per-action nags.
- MVP supports a single approval profile (`moderate`); strict/permissive profiles are post-MVP.

- Approvals are typed, hash-bound to immutable inputs, and time-bounded (TTL/expiry); stale approvals are invalid.
- Policy decisions and failures use a shared protocol error envelope and stable reason codes.

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`
- Product alignment: Centralizes least-privilege enforcement and “no escalation-in-place”.

## Standards Applied

- None yet.
