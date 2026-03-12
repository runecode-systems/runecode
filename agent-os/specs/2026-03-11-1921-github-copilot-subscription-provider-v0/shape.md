# GitHub Copilot Subscription Provider (Official Runtime Bridge) — Shaping Notes

## Scope

Add a post-MVP provider integration that uses a user's GitHub Copilot subscription while preserving RuneCode's strict isolation and audit guarantees.

## Decisions

- This provider is post-MVP; MVP uses API-key based providers only.
- Use only official Copilot mechanisms; no credential emulation.
- `secretsd` is the only long-lived secrets store.
- No environment-variable token injection.
- The Copilot runtime executes under `model-gateway` in an "LLM-only" mode (deny tool/file operations).
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
