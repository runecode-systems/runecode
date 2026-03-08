# Launcher MicroVM Backend v0 — Shaping Notes

## Scope

Implement the microVM isolation backend needed for MVP (Linux-first), while keeping a cross-platform design.

## Decisions

- MicroVMs are the preferred/primary boundary.
- MVP uses vsock-first on Linux with a virtio-serial fallback, with mandatory message-level authentication+encryption (do not rely on transport properties).
- MicroVM failure must not auto-enable container mode.
- QEMU hardening/sandboxing is part of the MVP security boundary (not a later polish item).

## Context

- Visuals: None.
- References: `agent-os/product/tech-stack.md`
- Product alignment: Strong isolation boundary and explicit artifact movement.

## Standards Applied

- None yet.
