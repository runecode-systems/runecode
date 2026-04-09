## Summary
RuneCode can launch and manage isolated Linux-first microVM-based roles with a clear auditable isolation boundary, per-session isolate identity keys, no host filesystem mounts, and a small set of backend-neutral contracts that later container, attestation, macOS, and Windows work can reuse without changing core semantics.

## Problem
The current change captures the security direction, but several contract seams remain implicit:
- launcher vs broker ownership
- backend/runtime posture vocabulary
- handshake and session-binding object model
- image identity and attachment planning
- audit payload and backend error taxonomy

If those seams are left to the first Linux/QEMU implementation, QEMU-specific details and MVP shortcuts are likely to leak into later container, cross-platform, durable-state, and attestation work.

## Proposed Change
- Freeze a backend-neutral trusted interface for launch, attachment, hardening, session establishment, and terminal reporting.
- Make launcher/broker ownership explicit while keeping the logical trust-boundary contract as the broker local API.
- Separate `backend_kind`, runtime isolation assurance, provisioning/binding posture, and audit posture into explicit, non-overloaded concepts.
- Define typed session-establishment, runtime-image, attachment-plan, hardening-posture, audit-payload, and backend-error contracts before backend implementation expands.
- Keep operator-visible contracts topology-neutral and free of hypervisor flags, host paths, mutable image tags, or transport-specific identity.

## Why Now
This work remains scheduled for v0.1.0-alpha.3, and the first real isolated backend is on the critical path for the first end-to-end secure slice. Freezing these contracts now reduces later schema churn across container opt-in, runner durable state, macOS/Windows backends, and isolate attestation.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Shipping every follow-on backend, attestation, or signing feature during this change.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Treating QEMU flags, host-local paths, or other implementation details as the long-lived logical contract.

## Impact
This change becomes the contract-setting foundation for runtime isolation rather than only the first Linux/QEMU implementation. It should let later container, macOS, Windows, durable-state, and attestation work reuse the same object semantics, run surfaces, and audit posture without rewriting the core model.
