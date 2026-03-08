# Image/Toolchain Signing Pipeline — Shaping Notes

## Scope

Add a signed supply chain for isolate images and toolchains and enforce it at boot.

## Decisions

- Image/toolchain signing keys are separate from manifest signing.
- Enforcement is fail-closed.

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-launcher-microvm-backend-v0/`
- Product alignment: Reduces supply chain risk in the isolation boundary.

## Standards Applied

- None yet.
