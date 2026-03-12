# OpenAI ChatGPT Subscription Provider (OAuth + Codex Bridge) — Shaping Notes

## Scope

Add a post-MVP provider integration that uses a user's ChatGPT subscription for GPT model access while preserving RuneCode's strict isolation and audit guarantees.

## Decisions

- This provider is post-MVP; MVP uses API-key based providers only.
- RuneCode maintains its own official OAuth client registration.
- Auth egress and model egress are separated:
  - `auth-gateway` performs OAuth and refresh and has auth-only egress.
  - `model-gateway` performs model inference egress and has no workspace access.
- `secretsd` is the only long-lived secrets store; bridge runtimes must not persist credentials.
- No environment-variable token injection.
- RuneCode integrates only with officially supported, user-installed runtimes (no bundling/redistribution).
- External runtime identity/version are discovered and logged per request; contract tests validate the RPC surface.
- External runtime compatibility uses a "tested range" + compatibility probe so RuneCode does not require updates for every vendor release.
  - Untested-but-probe-passing versions require explicit user acknowledgment and are recorded as a degraded posture.
- Sessions are ephemeral by default; persisted conversation state requires explicit manifest+policy enablement.

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-secretsd-model-gateway-v0/`, `agent-os/specs/2026-03-08-1039-policy-engine-v0/`, `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`
- Product alignment: Enables subscription access without expanding the trust boundary to include workspace access + egress + long-lived secrets.

## Standards Applied

- None yet.
