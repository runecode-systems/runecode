# Windows MicroVM Runtime Support — Shaping Notes

## Scope

Add runtime microVM support on Windows while keeping the same policy, schema, and audit semantics.

## Decisions

- Runtime support is distinct from CI portability; CI comes first.
- Windows uses OS-appropriate local IPC and permissions.

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-launcher-microvm-backend-v0/`
- Product alignment: Cross-platform single security model.

## Standards Applied

- None yet.
