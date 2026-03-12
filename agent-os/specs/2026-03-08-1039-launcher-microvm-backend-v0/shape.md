# Launcher MicroVM Backend v0 — Shaping Notes

## Scope

Implement the microVM isolation backend needed for MVP (Linux-first), while keeping a cross-platform design.

## Decisions

- MicroVMs are the preferred/primary boundary.
- MVP uses vsock-first on Linux with a virtio-serial fallback, with mandatory message-level authentication+encryption (do not rely on transport properties).
- Isolate key provisioning is TOFU for MVP; binding context (image digest + handshake transcript hash) is recorded and surfaced as a degraded posture.
- MicroVM failure must not auto-enable container mode.
- QEMU hardening/sandboxing is part of the MVP security boundary (not a later polish item).
- Performance work (boot latency, warm pools, caching) must not relax isolation semantics or bypass audit/policy.
- Warm pools/caches must not introduce cross-run state bleed; reuse requires reset-to-clean (or destroy) semantics and verifiable, manifest-pinned artifacts.

- CI may not always have KVM; backend-agnostic components must be testable without KVM, while microVM e2e runs can use a dedicated KVM-capable lane.

## Context

- Visuals: None.
- References: `agent-os/product/tech-stack.md`
- Product alignment: Strong isolation boundary and explicit artifact movement.

## Standards Applied

- None yet.
