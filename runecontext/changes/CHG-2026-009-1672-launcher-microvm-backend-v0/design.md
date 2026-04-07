# Design

## Overview
Implement the Linux microVM isolation backend, including launcher hardening, guest-image contracts, artifact attachment, and fail-closed lifecycle handling.

## Key Decisions
- MicroVMs are the preferred/primary boundary.
- MVP uses vsock-first on Linux with a virtio-serial fallback, with mandatory message-level authentication+encryption (do not rely on transport properties).
- Isolate session identity keys are per-session ephemeral, generated inside the isolate boundary, and must prove possession during the handshake.
- Isolate key provisioning is TOFU for MVP; binding context (image digest + handshake transcript hash) is recorded and surfaced as a degraded posture.
- Hosting node identity is audit metadata, not isolate identity; the isolate key binding model must stay topology-neutral so future multi-node scheduling does not change object semantics.
- Later attestation work upgrades the same session-key model rather than replacing it with a different isolate identity contract.
- MicroVM failure must not auto-enable container mode.
- QEMU hardening/sandboxing is part of the MVP security boundary (not a later polish item).
- Performance work (boot latency, warm pools, caching) must not relax isolation semantics or bypass audit/policy.
- Warm pools/caches must not introduce cross-run state bleed; reuse requires reset-to-clean (or destroy) semantics and verifiable, manifest-pinned artifacts.
- CI may not always have KVM; backend-agnostic components must be testable without KVM, while microVM e2e runs can use a dedicated KVM-capable lane.
- Backend kind and assurance posture are first-class operator-visible outputs that should align with shared broker run-summary/run-detail surfaces rather than existing only as audit side notes.

## Main Workstreams
- MicroVM Backend Architecture
- QEMU Hardening / Host Sandbox (MVP)
- Guest Image + Boot Contract (Minimal)
- Disk + Artifact Attachment Model
- Resource Limits + Lifecycle
- Failure Handling

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
