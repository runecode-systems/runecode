# Design

## Overview
Implement the Linux-first microVM isolation backend, including launcher hardening, guest-image contracts, artifact attachment, fail-closed lifecycle handling, and the small set of typed contracts that later container, attestation, macOS, and Windows work should reuse without changing core runtime semantics.

## Key Decisions
- MicroVMs are the preferred/primary boundary.
- The launcher becomes a long-lived trusted runtime daemon/service under broker control.
- The broker remains the control-plane authority for policy, approvals, artifact authorization, and authoritative run/read-model state.
- The launcher remains the runtime authority for backend realization, hardening, attachment materialization, session establishment, watchdogs, and fail-closed termination.
- The logical trust-boundary contract remains the broker local API. Launcher-managed vsock/virtio-serial plumbing and session establishment are implementation details for carrying the same broker-mediated semantics, not a second runtime API.
- Define a backend-neutral trusted interface early and keep it small.
- MVP uses vsock-first on Linux with a virtio-serial fallback, with mandatory message-level authentication+encryption (do not rely on transport properties).
- Isolate session identity keys are per-session ephemeral, generated inside the isolate boundary, and must prove possession during the handshake.
- Isolate key provisioning is TOFU for MVP; binding context (image digest + handshake transcript hash) is recorded and surfaced as a degraded posture.
- Hosting node identity is audit metadata, not isolate identity; the isolate key binding model must stay topology-neutral so future multi-node scheduling does not change object semantics.
- Later attestation work upgrades the same session-key model rather than replacing it with a different isolate identity contract.
- Keep hypervisor flags, host-local filesystem paths, device numbering, guest mount paths, and transport allocation details out of boundary-visible or cross-backend logical contracts.
- Operator-visible runtime posture is split into separate axes:
  - `backend_kind` identifies the selected backend class (`microvm`, later `container`)
  - `assurance_level` should refer only to runtime isolation assurance, or be renamed to `isolation_assurance_level` when schemas can evolve cleanly
  - provisioning/binding posture remains separate (`tofu` in MVP, later attested variants)
  - audit verification posture remains separate
- Keep backend kind operator-facing and topology-neutral (`microvm`, not `qemu`). Hypervisor implementation details such as `qemu` and acceleration details such as `kvm`, `hvf`, or `whpx` belong in detailed runtime/hardening evidence rather than public run identity.
- The broker should expose authoritative runtime state as a projection of launcher-produced facts rather than inferring backend posture indirectly from audit-only or runner-only data.
- One isolate maps to one `role_instance`, one `role_kind`, and one `role_family` in MVP.
- MicroVM failure must not auto-enable container mode.
- QEMU hardening/sandboxing is part of the MVP security boundary (not a later polish item).
- Performance work (boot latency, warm pools, caching) must not relax isolation semantics or bypass audit/policy.
- Warm pools/caches must not introduce cross-run state bleed; reuse requires reset-to-clean (or destroy) semantics and verifiable, manifest-pinned artifacts.
- CI may not always have KVM; backend-agnostic components must be testable without KVM, while microVM e2e runs can use a dedicated KVM-capable lane.
- Backend kind and runtime isolation assurance are first-class operator-visible outputs that should align with shared broker run-summary/run-detail surfaces rather than existing only as audit side notes.

## Control-Plane Ownership
- Broker responsibilities:
  - policy evaluation and approval handling
  - artifact authorization and data-class enforcement
  - authoritative run state and operator-facing read models
  - projecting launcher-produced facts into `RunSummary` and `RunDetail`
- Launcher responsibilities:
  - backend realization
  - hardening and confinement
  - attachment materialization
  - session establishment and binding validation
  - watchdogs, timeouts, and fail-closed termination
- The broker may request a backend kind; the launcher may realize it or fail closed. The launcher must never silently substitute another backend.

## Foundational Contracts
The change should define a small trusted backend contract around typed objects such as:
- `BackendLaunchSpec`: broker-authored, policy-authorized launch intent for one role instance.
- `LaunchContext`: immutable read-only guest-visible context bound into launch/session identity.
- `AttachmentPlan`: authorized runtime attachments keyed by logical role instead of hypervisor-specific disk numbering.
- `RuntimeImageDescriptor`: digest-addressed guest image descriptor pinned in manifests.
- `BackendLaunchReceipt`: launcher-produced record of what backend was realized and what image inputs were used.
- `IsolateSessionBinding`: per-session isolate identity binding pinned to `{run_id, isolate_id, session_id}` with TOFU/attestation posture.
- `AppliedHardeningPosture`: requested posture, effective posture, and degraded reasons for runtime sandboxing.
- `BackendTerminalReport`: typed terminal outcome/failure report for lifecycle, audit, and operator surfaces.

These names may ship as protocol objects, trusted internal structs, or both, but the logical seams should be frozen early and remain backend-neutral.

## Runtime Posture Model
- Keep operator-visible posture as separate axes:
  - `backend_kind` identifies the selected backend class (`microvm`, later `container`)
  - `assurance_level` should refer only to runtime isolation assurance, or be renamed to `isolation_assurance_level` when schemas can evolve cleanly
  - provisioning/binding posture remains separate (`tofu` in MVP, later attested variants)
  - audit verification posture remains separate
- Keep `backend_kind` operator-facing and topology-neutral (`microvm`, not `qemu`).
- Hypervisor implementation details such as `qemu` and acceleration details such as `kvm`, `hvf`, or `whpx` belong in detailed runtime/hardening evidence rather than public run identity.
- The broker should expose authoritative runtime state as a projection of launcher-produced facts rather than inferring backend posture indirectly from audit-only or runner-only data.

## Session Establishment Model
- MVP uses vsock-first on Linux with a `virtio-serial` fallback, but transport choice must not change message or identity semantics.
- Session establishment should use a small typed handshake family such as:
  - `HostHello`
  - `IsolateHello`
  - `SessionReady`
- Ordinary broker API traffic flows only after this secure session is established.
- The durable isolate identity is the per-session isolate signing key generated inside the isolate boundary.
- Transport/session keys are channel keys only and must not become the durable identity model.
- The launcher generates `session_nonce`.
- The handshake records canonical transcript bytes and a `handshake_transcript_hash`.
- Replay detection, message framing, and strict size limits are required.
- TOFU is the MVP provisioning mode; later attestation work upgrades the same session-key model rather than replacing it with a different identity contract.

## Image and Attachment Model
- Manifests should identify guest images through one digest-addressed `RuntimeImageDescriptor` rather than mutable tags, host paths, or loose file references.
- The descriptor should carry:
  - backend/platform compatibility
  - boot-contract version
  - kernel/rootfs/initrd or equivalent component digests
  - optional signing metadata
  - future attestation/measurement hooks
- Launch/audit receipts should record both the descriptor digest and the concrete component digests actually used at boot.
- Runtime data movement should use a typed `AttachmentPlan` keyed by logical attachment role rather than hypervisor-specific device numbering.
- MVP attachment roles should include:
  - `launch_context` read-only
  - `workspace` run-scoped read-write encrypted volume
  - `input_artifacts` read-only immutable inputs
  - `scratch` ephemeral read-write working state
- No host filesystem mounts are permitted into the isolate.
- Guest mount locations remain backend/guest convention rather than public contract identity.

## Hardening and Failure Model
- Treat the QEMU process as part of the attack surface and harden/confine it by default.
- Record hardening posture as a typed object with requested posture, effective posture, and degraded reasons.
- Common posture fields should remain backend-neutral and cover:
  - execution identity posture
  - filesystem exposure posture
  - network exposure posture
  - syscall filtering posture
  - device surface posture
  - control-channel kind
  - acceleration kind
- Backend-specific evidence may add QEMU provenance, sandbox/profile identifiers, and similar implementation details without leaking host-local paths.
- Error handling should use a stable backend error taxonomy; launch/runtime failures must not be represented only through logs, exit codes, or hypervisor stderr parsing.

## Audit and Operator Surfaces
- `isolate_session_started` and `isolate_session_bound` should have explicit payload schemas rather than only registry entries or test-local shapes.
- Audit payloads should remain small and reference-heavy, pointing to launch/session/image/hardening objects instead of duplicating host-local implementation details.
- Later attestation work should upgrade TOFU posture without requiring a format break for these audit event families.
- Backend kind, runtime isolation assurance, provisioning posture, and audit posture should all remain visible through shared broker run-summary/run-detail surfaces.

## Lifecycle
- Backend lifecycle should be explicit enough for launch/session debugging and durable state without overloading operator run lifecycle state. A minimal internal progression is:
  - `planned`
  - `launching`
  - `started`
  - `binding`
  - `active`
  - `terminating`
  - `terminated`

## Main Workstreams
- Backend Interface + Ownership Model
- Runtime Posture Vocabulary + Authoritative State
- MicroVM Backend Architecture
- Session Handshake + Binding Contract
- QEMU Hardening / Host Sandbox (MVP)
- Runtime Image Descriptor + Boot Contract
- Attachment Plan + Artifact Movement
- Resource Limits + Lifecycle
- Failure Handling + Error Taxonomy
- Audit Payloads + Run-Surface Alignment

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
