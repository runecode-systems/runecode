# Git Gateway (Commit/Push/PR) — Shaping Notes

## Scope

Add a dedicated gateway role for git operations with defense-in-depth verification and strict allowlists.

## Decisions

- Git egress is treated as high risk and is isolated behind a gateway.
- Outbound verification must match signed patch artifacts.

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-secretsd-model-gateway-v0/`
- Product alignment: Prevents combining workspace RW with git credentials and public egress.

## Standards Applied

- None yet.
