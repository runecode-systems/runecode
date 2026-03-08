# Container Backend v0 (Explicit Opt-In) — Shaping Notes

## Scope

Implement a hardened container-based isolation backend for hosts where microVMs are unavailable or for explicit user choice.

## Decisions

- Containers are never a silent fallback; they require explicit opt-in and acknowledgment.
- The active backend and its assurance level are treated as first-class audit data.
- Container networking is isolated by default (no egress); any allowed egress is enforced via explicit network namespace + firewall/proxy rules, not convention.

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-policy-engine-v0/`
- Product alignment: Preserves deny-by-default semantics while being honest about boundary strength.

## Standards Applied

- None yet.
